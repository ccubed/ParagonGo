package combat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// MobRank holds the computed metrics for a single mob spec, evaluated at
// the mob's effective level against a level-matched baseline opponent.
type MobRank struct {
	MobId   int
	Name    string
	Zone    string
	Level   int
	Hostile bool

	// Combat metrics at the mob's effective level.
	DiceRoll string
	DPS      float64
	MaxDmg   int
	HP       int
	Defense  int
	// EHP is raw HP divided by the fraction of damage that gets through:
	// EHP = HP / (1 - min(Defense/200, 0.95))
	EHP    float64
	Threat float64 // DPS * EHP

	// Loot fields from the raw spec (level-independent).
	Gold           int
	ItemCount      int
	ItemDropChance int
	ItemValue      int
	// LootScore = Gold + (ItemDropChance/100 * ItemValue)
	// ItemValue includes carried items and worn items (excluding CanNeverBeRemoved).
	LootScore float64

	// Aggregate scores used for tab sorting.
	ThreatScore  float64 // = Threat
	DefenseScore float64 // = EHP
	OffenseScore float64 // = DPS
}

// baselineCharacterAtLevel returns a neutral character whose six stats are
// scaled to the given level: stat = clamp(level*2, 1, 100).
func baselineCharacterAtLevel(level int) characters.Character {
	stat := level * 2
	if stat < 1 {
		stat = 1
	}
	if stat > 100 {
		stat = 100
	}
	hp := 50 + level*5
	c := characters.Character{
		Level:  level,
		Health: hp,
	}
	c.Stats = stats.Statistics{
		Strength:   stats.StatInfo{Base: stat},
		Speed:      stats.StatInfo{Base: stat},
		Smarts:     stats.StatInfo{Base: stat},
		Vitality:   stats.StatInfo{Base: stat},
		Mysticism:  stats.StatInfo{Base: stat},
		Perception: stats.StatInfo{Base: stat},
	}
	c.RecalculateStats()
	return c
}

// RankMobs computes ranking metrics for every loaded mob spec at each mob's
// effective level and returns slices sorted by three criteria:
//
//   - byThreat   – DPS × eHP
//   - byLoot     – expected loot value per kill
//   - byDefense  – effective HP
func RankMobs() (byThreat, byLoot, byDefense []MobRank) {
	allSpecs := mobs.GetAllMobSpecs()

	ranks := make([]MobRank, 0, len(allSpecs))

	for _, spec := range allSpecs {
		// Determine the effective level for this mob.
		// If the zone has auto-scaling, use the midpoint of the scale range.
		// Otherwise use the mob's defined level.
		effectiveLevel := spec.Character.Level
		if zCfg := rooms.GetZoneConfig(spec.Zone); zCfg != nil && zCfg.MobAutoScale.Maximum > 0 {
			effectiveLevel = (zCfg.MobAutoScale.Minimum + zCfg.MobAutoScale.Maximum) / 2
		}
		if effectiveLevel < 1 {
			effectiveLevel = 1
		}

		baseline := baselineCharacterAtLevel(effectiveLevel)

		rank := MobRank{
			MobId:          int(spec.MobId),
			Name:           spec.Character.Name,
			Zone:           spec.Zone,
			Level:          effectiveLevel,
			Hostile:        spec.Hostile,
			Gold:           spec.Character.Gold,
			ItemDropChance: spec.ItemDropChance,
		}

		// Loot: sum item spec values for all carried items.
		for _, item := range spec.Character.Items {
			if iSpec := items.GetItemSpec(item.ItemId); iSpec != nil {
				rank.ItemValue += iSpec.Value
				rank.ItemCount++
			}
		}
		// Worn items also drop (subject to ItemDropChance), unless CanNeverBeRemoved.
		for _, item := range spec.Character.Equipment.GetAllItems() {
			if item.IsRemoveLocked() {
				continue
			}
			if iSpec := items.GetItemSpec(item.ItemId); iSpec != nil {
				rank.ItemValue += iSpec.Value
				rank.ItemCount++
			}
		}
		rank.LootScore = float64(rank.Gold) + float64(rank.ItemDropChance)/100.0*float64(rank.ItemValue)

		// Combat snapshot at the mob's effective level.
		mob, err := newSimMob(spec.MobId, effectiveLevel)
		if err == nil {
			rank.DPS = expectedDPS(mob.Character, baseline)

			// Dice roll and max damage.
			var attacks, dCount, dSides, dBonus int
			if mob.Character.Equipment.Weapon.ItemId > 0 {
				wSpec := mob.Character.Equipment.Weapon.GetSpec()
				rank.DiceRoll = wSpec.Damage.DiceRoll
				attacks, dCount, dSides, dBonus, _ = mob.Character.Equipment.Weapon.GetDiceRoll()
			} else {
				attacks, dCount, dSides, dBonus, _ = mob.Character.GetDefaultDiceRoll()
				if dCount > 0 && dSides > 0 {
					rank.DiceRoll = fmt.Sprintf("%dd%d", dCount, dSides)
					if dBonus != 0 {
						rank.DiceRoll += fmt.Sprintf("%+d", dBonus)
					}
					if attacks > 1 {
						rank.DiceRoll = fmt.Sprintf("%d@%s", attacks, rank.DiceRoll)
					}
				}
			}
			if attacks < 1 {
				attacks = 1
			}
			if rawMax := attacks * (dCount*dSides + dBonus); rawMax > 0 {
				rank.MaxDmg = rawMax
			}

			rank.HP = mob.Character.HealthMax.Value
			rank.Defense = mob.Character.GetDefense()

			defFrac := float64(rank.Defense) / 200.0
			if defFrac > 0.95 {
				defFrac = 0.95
			}
			if defFrac < 0 {
				defFrac = 0
			}
			rank.EHP = float64(rank.HP) / (1.0 - defFrac)
			rank.Threat = rank.DPS * rank.EHP
		}

		rank.ThreatScore = rank.Threat
		rank.DefenseScore = rank.EHP
		rank.OffenseScore = rank.DPS

		ranks = append(ranks, rank)
	}

	byThreat = make([]MobRank, len(ranks))
	copy(byThreat, ranks)
	sort.Slice(byThreat, func(i, j int) bool {
		return byThreat[i].ThreatScore > byThreat[j].ThreatScore
	})

	byLoot = make([]MobRank, len(ranks))
	copy(byLoot, ranks)
	sort.Slice(byLoot, func(i, j int) bool {
		return byLoot[i].LootScore > byLoot[j].LootScore
	})

	byDefense = make([]MobRank, len(ranks))
	copy(byDefense, ranks)
	sort.Slice(byDefense, func(i, j int) bool {
		return byDefense[i].DefenseScore > byDefense[j].DefenseScore
	})

	return byThreat, byLoot, byDefense
}

// FormatMobRankings returns a human-readable table of all three ranking views.
func FormatMobRankings() string {
	byThreat, byLoot, byDefense := RankMobs()

	var sb strings.Builder

	writeTable := func(title string, rows []MobRank, scoreLabel string, score func(MobRank) string) {
		fmt.Fprintf(&sb, "\n=== %s ===\n", title)
		fmt.Fprintf(&sb, "%-4s %-5s %-28s %-16s %-8s %-8s %-8s %-8s %s\n",
			"Rank", "ID", "Name", "Zone", "Hostile", "DPS", "EHP", "LootScore", scoreLabel)
		fmt.Fprintln(&sb, strings.Repeat("-", 110))
		for i, r := range rows {
			hostile := ""
			if r.Hostile {
				hostile = "yes"
			}
			fmt.Fprintf(&sb, "%-4d %-5d %-28s %-16s %-8s %-8.2f %-8.1f %-8.1f %s\n",
				i+1, r.MobId, r.Name, r.Zone, hostile,
				r.DPS, r.EHP, r.LootScore, score(r))
		}
	}

	writeTable("Ranked by Threat (DPS × eHP)", byThreat, "Threat", func(r MobRank) string {
		return fmt.Sprintf("%.2f", r.ThreatScore)
	})
	writeTable("Ranked by Loot Value", byLoot, "LootScore", func(r MobRank) string {
		return fmt.Sprintf("%.1f", r.LootScore)
	})
	writeTable("Ranked by Defense (eHP)", byDefense, "DefScore", func(r MobRank) string {
		return fmt.Sprintf("%.1f", r.DefenseScore)
	})

	return sb.String()
}
