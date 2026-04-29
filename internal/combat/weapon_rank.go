package combat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// WeaponRank holds the computed metrics for a single weapon spec.
type WeaponRank struct {
	ItemId   int
	Name     string
	Subtype  items.ItemSubType
	Hands    int
	DiceRoll string

	// Raw dice metrics (weapon only, no character stat bonuses)
	AvgDmg float64
	MaxDmg int

	// DPR is the average damage that would land per round if every attack
	// connected with no misses, dodges, or defense reduction. It is the
	// raw per-round ceiling: avgDmgPerHit × attackCount.
	DPR float64

	// AdjDPR applies the same two-handed and wait-round opportunity-cost
	// penalties as AdjDPS, but to the raw DPR rather than the
	// combat-engine DPS.
	AdjDPR float64

	// expectedDPS from the combat engine against a neutral opponent
	// at equal stats (no character advantage either way).
	DPS float64

	// DPS adjusted for opportunity cost: two-handed weapons and
	// weapons with wait rounds give up something (offhand slot /
	// attack frequency), so we penalise them proportionally.
	// WaitRounds > 0 means the weapon skips that many rounds
	// between attacks; Hands == 2 means no offhand.
	AdjDPS float64

	WaitRounds int
}

// baselineCharacter returns a neutral character with equal stats in all
// six attributes so that stat-delta functions return their minimum values
// and the comparison is purely weapon-driven.
func baselineCharacter() characters.Character {
	const stat = 50
	c := characters.Character{
		Level:  10,
		Health: 100,
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

// RankWeapons computes ranking metrics for every loaded weapon spec and
// returns slices sorted by three different criteria:
//
//   - byDPS      – raw expected DPS against an equal-stat opponent
//   - byAdjDPS   – DPS penalised for two-handed / wait-round cost
//   - byMaxDmg   – theoretical maximum single-hit damage
//
// All three slices contain the same entries; only the order differs.
//
// The function is intended for design-time balance analysis and is not
// called during normal gameplay.
func RankWeapons() (byDPS, byAdjDPS, byMaxDmg []WeaponRank) {
	attacker := baselineCharacter()
	defender := baselineCharacter()

	allSpecs := items.GetAllItemSpecs()

	ranks := make([]WeaponRank, 0, len(allSpecs))

	for _, spec := range allSpecs {
		if spec.Type != items.Weapon {
			continue
		}

		dmg := spec.Damage
		if dmg.DiceCount == 0 && dmg.SideCount == 0 {
			continue
		}

		attacks := dmg.Attacks
		if attacks < 1 {
			attacks = 1
		}

		avgDmg := float64(attacks) * (float64(dmg.DiceCount)*float64(dmg.SideCount+1)/2.0 + float64(dmg.BonusDamage))
		maxDmg := attacks * (dmg.DiceCount*dmg.SideCount + dmg.BonusDamage)

		// Equip the weapon on the attacker and compute DPS.
		attacker.Equipment.Weapon = items.New(spec.ItemId)
		attacker.Equipment.Offhand = items.Item{}
		attacker.RecalculateStats()

		dps := expectedDPS(attacker, defender)

		// DPR: raw average damage per round if every attack connects —
		// no hit-chance, dodge, or defense reduction applied.
		// Uses the same attack count the combat engine would use.
		atkCount := combatAttackCount(attacker, defender)
		avgPerHit := float64(dmg.DiceCount)*float64(dmg.SideCount+1)/2.0 + float64(dmg.BonusDamage)
		if avgPerHit < 0 {
			avgPerHit = 0
		}
		dpr := float64(attacks) * float64(atkCount) * avgPerHit

		// Adjusted DPS / DPR:
		//   - Two-handed weapons block the offhand slot. We model the
		//     opportunity cost as if the player could otherwise wield an
		//     identical one-handed version, so we apply a 0.75 factor
		//     (giving up ~25% of potential dual-wield upside).
		//   - WaitRounds reduces effective attack frequency. A weapon
		//     with waitRounds W fires once every (1 + W) rounds, so the
		//     effective DPS multiplier is 1/(1+W).
		adjDPS := dps
		adjDPR := dpr
		if spec.Hands == items.TwoHanded {
			adjDPS *= 0.75
			adjDPR *= 0.75
		}
		if spec.WaitRounds > 0 {
			adjDPS /= float64(1 + spec.WaitRounds)
			adjDPR /= float64(1 + spec.WaitRounds)
		}

		ranks = append(ranks, WeaponRank{
			ItemId:     spec.ItemId,
			Name:       spec.Name,
			Subtype:    spec.Subtype,
			Hands:      spec.Hands,
			DiceRoll:   dmg.DiceRoll,
			AvgDmg:     avgDmg,
			MaxDmg:     maxDmg,
			DPR:        dpr,
			AdjDPR:     adjDPR,
			DPS:        dps,
			AdjDPS:     adjDPS,
			WaitRounds: spec.WaitRounds,
		})
	}

	// Reset attacker equipment when done.
	attacker.Equipment.Weapon = items.Item{}

	byDPS = make([]WeaponRank, len(ranks))
	copy(byDPS, ranks)
	sort.Slice(byDPS, func(i, j int) bool {
		return byDPS[i].DPS > byDPS[j].DPS
	})

	byAdjDPS = make([]WeaponRank, len(ranks))
	copy(byAdjDPS, ranks)
	sort.Slice(byAdjDPS, func(i, j int) bool {
		return byAdjDPS[i].AdjDPS > byAdjDPS[j].AdjDPS
	})

	byMaxDmg = make([]WeaponRank, len(ranks))
	copy(byMaxDmg, ranks)
	sort.Slice(byMaxDmg, func(i, j int) bool {
		return byMaxDmg[i].MaxDmg > byMaxDmg[j].MaxDmg
	})

	return byDPS, byAdjDPS, byMaxDmg
}

// FormatWeaponRankings returns a human-readable table of all three
// ranking views, suitable for logging or printing to a terminal.
func FormatWeaponRankings() string {
	byDPS, byAdjDPS, byMaxDmg := RankWeapons()

	var sb strings.Builder

	writeTable := func(title string, rows []WeaponRank, scoreLabel string, score func(WeaponRank) string) {
		fmt.Fprintf(&sb, "\n=== %s ===\n", title)
		fmt.Fprintf(&sb, "%-4s %-32s %-10s %-8s %-5s %-5s %s\n",
			"Rank", "Name", "Dice", "Subtype", "Hands", "Wait", scoreLabel)
		fmt.Fprintln(&sb, strings.Repeat("-", 90))
		for i, r := range rows {
			fmt.Fprintf(&sb, "%-4d %-32s %-10s %-8s %-5d %-5d %s\n",
				i+1, r.Name, r.DiceRoll, string(r.Subtype), r.Hands, r.WaitRounds, score(r))
		}
	}

	writeTable("Ranked by Raw DPS", byDPS, "DPS", func(r WeaponRank) string {
		return fmt.Sprintf("%.3f", r.DPS)
	})
	writeTable("Ranked by Adjusted DPS (2H / wait-round penalty)", byAdjDPS, "AdjDPS", func(r WeaponRank) string {
		return fmt.Sprintf("%.3f", r.AdjDPS)
	})
	writeTable("Ranked by Max Single-Hit Damage", byMaxDmg, "MaxDmg", func(r WeaponRank) string {
		return fmt.Sprintf("%d  (avg %.2f)", r.MaxDmg, r.AvgDmg)
	})

	return sb.String()
}
