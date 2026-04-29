package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// statDelta returns the fraction of the configured range that the attacker
// earns over the defender, clamped to [0, 1].
// Formula: clamp(max(0, atkStat - defStat), 0, 100) / 100
func statDelta(atkStat, defStat int) float64 {
	delta := float64(atkStat - defStat)
	if delta < 0 {
		delta = 0
	}
	if delta > 100 {
		delta = 100
	}
	return delta / 100.0
}

// resolveAttackWeapons returns the candidate weapon list for a character,
// applying the same selection logic used by calculateCombat.
// It does not trim for dual-wield skill — callers handle that themselves.
func resolveAttackWeapons(char characters.Character) []items.Item {
	attackWeapons := []items.Item{}
	if char.Equipment.Weapon.ItemId > 0 {
		attackWeapons = append(attackWeapons, char.Equipment.Weapon)
	}
	if char.Equipment.Offhand.ItemId > 0 && char.Equipment.Offhand.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, char.Equipment.Offhand)
	}
	if len(attackWeapons) == 0 {
		attackWeapons = append(attackWeapons, items.Item{ItemId: 0})
	}
	return attackWeapons
}

// damageBonus returns the flat bonus damage an attacker earns over a defender
// based on the Strength stat delta and the configured bounds.
func damageBonus(atkStr, defStr int) int {
	cfg := configs.GetCombatConfig()
	minBonus := int(cfg.DamageBonusMin)
	maxBonus := int(cfg.DamageBonusMax)
	actual := int(math.Floor(statDelta(atkStr, defStr) * float64(maxBonus)))
	if actual < minBonus {
		actual = minBonus
	}
	return actual
}

// hitChance returns a hit probability in [ToHitMin, ToHitMax] based on the
// Speed stat delta between attacker and defender.
func hitChance(atkSpd, defSpd int) int {
	cfg := configs.GetCombatConfig()
	minHit := int(cfg.ToHitMin)
	maxHit := int(cfg.ToHitMax)
	actual := int(math.Floor(statDelta(atkSpd, defSpd) * float64(maxHit)))
	if actual < minHit {
		actual = minHit
	}
	return actual
}

// Hits returns whether an attack connects, incorporating an optional modifier.
func Hits(atkSpd, defSpd, hitModifier int) bool {
	toHit := hitChance(atkSpd, defSpd)
	toHit += hitModifier

	cfg := configs.GetCombatConfig()
	minHit := int(cfg.ToHitMin)
	maxHit := int(cfg.ToHitMax)
	if toHit < minHit {
		toHit = minHit
	}
	if toHit > maxHit {
		toHit = maxHit
	}

	hitRoll := util.Rand(100)
	util.LogRoll(`Hits`, hitRoll, toHit)
	return hitRoll < toHit
}

// extraAttackCount returns the number of bonus attacks for weaponless/claws
// combat based on the Speed stat delta and the configured bounds.
func extraAttackCount(atkSpd, defSpd int) int {
	cfg := configs.GetCombatConfig()
	minExtra := int(cfg.ExtraAttacksMin)
	maxExtra := int(cfg.ExtraAttacksMax)
	actual := int(math.Floor(statDelta(atkSpd, defSpd) * float64(maxExtra)))
	if actual < minExtra {
		actual = minExtra
	}
	return actual
}

// weaponlessAttackCount returns the total attack count (1 base + extra) for
// unarmed or claws combat.
func weaponlessAttackCount(atkSpd, defSpd, attacksMod int) int {
	count := 1 + extraAttackCount(atkSpd, defSpd)
	count += attacksMod
	if count < 1 {
		count = 1
	}
	return count
}

// combatAttackCount returns the attack count for the round. For weaponless or
// claws attacks the extra-attack formula applies; armed attacks always yield 1.
func combatAttackCount(sourceChar characters.Character, targetChar characters.Character) int {
	weapons := resolveAttackWeapons(sourceChar)
	isWeaponless := len(weapons) == 1 && weapons[0].ItemId == 0
	isClaws := len(weapons) == 1 && weapons[0].ItemId > 0 && weapons[0].GetSpec().Subtype == items.Claws

	if isWeaponless || isClaws {
		return weaponlessAttackCount(
			sourceChar.Stats.Speed.ValueAdj,
			targetChar.Stats.Speed.ValueAdj,
			sourceChar.StatMod(`attacks`),
		)
	}
	return 1
}

// critChance returns the integer crit probability in [CritChanceMin,
// CritChanceMax] based on the Smarts stat delta. Buff flags are applied after.
func critChance(atkSmarts, defSmarts int, hasAccuracy, targetHasBlink bool) int {
	cfg := configs.GetCombatConfig()
	minChance := int(cfg.CritChanceMin)
	maxChance := int(cfg.CritChanceMax)
	actual := int(math.Floor(statDelta(atkSmarts, defSmarts) * float64(maxChance)))
	if actual < minChance {
		actual = minChance
	}
	if hasAccuracy {
		actual *= 2
	}
	if targetHasBlink {
		actual /= 2
	}
	if actual < minChance {
		actual = minChance
	}
	if actual > 100 {
		actual = 100
	}
	return actual
}

// Crits rolls whether an attack is a critical hit.
func Crits(sourceChar characters.Character, targetChar characters.Character) bool {
	chance := critChance(
		sourceChar.Stats.Smarts.ValueAdj,
		targetChar.Stats.Smarts.ValueAdj,
		sourceChar.HasBuffFlag(buffs.Accuracy),
		targetChar.HasBuffFlag(buffs.Blink),
	)
	critRoll := util.Rand(100)
	util.LogRoll(`Crits`, critRoll, chance)
	return critRoll < chance
}

// critMultiplier returns the damage multiplier for a critical hit in
// [CritMultMin, CritMultMax] based on the Perception stat delta.
func critMultiplier(atkPerc, defPerc int) float64 {
	cfg := configs.GetCombatConfig()
	minMult := float64(cfg.CritMultMin)
	maxMult := float64(cfg.CritMultMax)
	actual := statDelta(atkPerc, defPerc) * maxMult
	if actual < minMult {
		actual = minMult
	}
	return actual
}

// critDamageBonus returns the extra damage added to a hit that is a critical,
// scaled by the attacker's crit multiplier relative to the defender.
func critDamageBonus(dCount, dSides, dBonus, atkPerc, defPerc int) int {
	base := dCount*dSides + dBonus
	if base < 0 {
		base = 0
	}
	mult := critMultiplier(atkPerc, defPerc)
	return int(math.Floor(float64(base) * (mult - 1.0)))
}

// dodgeChance returns the probability in [DodgeChanceMin, DodgeChanceMax] that
// the defender dodges an incoming hit, based on the defender's Perception
// advantage over the attacker.
func dodgeChance(defPerc, atkPerc int) int {
	cfg := configs.GetCombatConfig()
	minDodge := int(cfg.DodgeChanceMin)
	maxDodge := int(cfg.DodgeChanceMax)
	actual := int(math.Floor(statDelta(defPerc, atkPerc) * float64(maxDodge)))
	if actual < minDodge {
		actual = minDodge
	}
	return actual
}

// Dodges returns true when the defender successfully dodges an attack.
func Dodges(defPerc, atkPerc int) bool {
	chance := dodgeChance(defPerc, atkPerc)
	roll := util.Rand(100)
	util.LogRoll(`Dodges`, roll, chance)
	return roll < chance
}

// dualWieldHitPenalty returns the negative hit modifier applied to the offhand
// weapon based on the attacker's dual-wield skill level.
func dualWieldHitPenalty(dwLevel int) int {
	if dwLevel < 4 {
		return -35
	}
	return -25
}

// dualWieldActiveWeaponCount returns how many weapons fire this round based on
// dual-wield skill level and whether both equipped weapons are claws.
func dualWieldActiveWeaponCount(dwLevel int, bothClaws bool) int {
	if bothClaws || dwLevel >= 3 {
		return 2
	}
	if dwLevel == 2 {
		if util.Rand(100) < 50 {
			return 2
		}
		return 1
	}
	return 1
}

// applyDefenseReduction applies a stochastic defense roll to incoming damage,
// returning the final damage and the amount reduced.
func applyDefenseReduction(damage, defenseRating int) (finalDamage, reduction int) {
	defenseAmt := util.Rand(defenseRating)
	if defenseAmt > 0 {
		reduction = int(math.Round((float64(defenseAmt) / 100) * float64(damage)))
		finalDamage = damage - reduction
		return finalDamage, reduction
	}
	return damage, 0
}

// damagePercentOfMax returns the dealt damage expressed as a percentage of the
// theoretical maximum damage for the given dice configuration.
func damagePercentOfMax(damage, dCount, dSides, dBonus int) int {
	maxDmg := dCount*dSides + dBonus
	if maxDmg < 1 {
		maxDmg = 1
	}
	return int(math.Ceil(float64(damage) / float64(maxDmg) * 100))
}

// tameSizeModifier returns the taming chance modifier based on the tamer's size.
func tameSizeModifier(size races.Size) int {
	switch size {
	case races.Large:
		return -25
	case races.Small:
		return 0
	default:
		return -10
	}
}

// tameHealthBonus returns the taming bonus granted when the tamer is injured.
// A tamer at full HP contributes 0; a tamer near death contributes up to 50.
func tameHealthBonus(currentHP, maxHP int) float64 {
	return 50 - math.Ceil(float64(currentHP)/float64(maxHP)*50)
}

// chanceToTame computes the raw taming probability from primitive parameters.
func chanceToTame(
	proficiency int,
	levelDiff int,
	currentHP int,
	maxHP int,
	tamerSize races.Size,
	targetIsAggro bool,
) int {
	const (
		modSkillMin     = 1
		modSkillMax     = 100
		modLevelDiffMin = -25
		modLevelDiffMax = 25
		factorIsAggro   = 0.50
	)

	if proficiency < modSkillMin {
		proficiency = modSkillMin
	} else if proficiency > modSkillMax {
		proficiency = modSkillMax
	}

	if levelDiff > modLevelDiffMax {
		levelDiff = modLevelDiffMax
	} else if levelDiff < modLevelDiffMin {
		levelDiff = modLevelDiffMin
	}

	sizeModifier := tameSizeModifier(tamerSize)
	healthModifier := tameHealthBonus(currentHP, maxHP)

	aggroModifier := 1.0
	if targetIsAggro {
		aggroModifier = factorIsAggro
	}

	return int(math.Ceil((float64(proficiency) + float64(levelDiff) + healthModifier + float64(sizeModifier)) * aggroModifier))
}

// ChanceToTame returns the probability that a user successfully tames a mob.
func ChanceToTame(s *users.UserRecord, t *mobs.Mob) int {
	raceInfo := races.GetRace(s.Character.GetRaceId())
	return chanceToTame(
		s.Character.MobMastery.GetTame(int(t.MobId)),
		s.Character.Level-t.Character.Level,
		s.Character.Health,
		s.Character.HealthMax.Value,
		raceInfo.Size,
		t.Character.IsAggro(s.UserId, 0),
	)
}

// AlignmentChange returns the alignment delta for a killer after slaying a target.
func AlignmentChange(killerAlignment int8, killedAlignment int8) int {

	isKillerGood := killerAlignment > characters.AlignmentNeutralHigh
	isKillerEvil := killerAlignment < characters.AlignmentNeutralLow
	isKillerNeutral := killerAlignment >= characters.AlignmentNeutralLow && killerAlignment <= characters.AlignmentNeutralHigh

	isKilledGood := killedAlignment > characters.AlignmentNeutralHigh
	isKilledEvil := killedAlignment < characters.AlignmentNeutralLow
	isKilledNeutral := killedAlignment >= characters.AlignmentNeutralLow && killedAlignment <= characters.AlignmentNeutralHigh

	deltaAbs := math.Abs(math.Max(float64(killerAlignment), float64(killedAlignment))-math.Min(float64(killerAlignment), float64(killedAlignment))) * 0.5

	changeAmt := 0
	if deltaAbs <= 10 {
		changeAmt = 0
	} else if deltaAbs <= 30 {
		changeAmt = 1
	} else if deltaAbs <= 60 {
		changeAmt = 2
	} else if deltaAbs <= 80 {
		changeAmt = 3
	} else {
		changeAmt = 4
	}

	factor := 0

	if isKillerGood {
		if isKilledGood {
			factor = -2
			changeAmt = int(math.Max(float64(changeAmt), 1))
		} else if isKilledEvil {
			factor = 1
		} else if isKilledNeutral {
			factor = -1
		}
	} else if isKillerEvil {
		if isKilledGood {
			factor = -1
		} else if isKilledEvil {
			factor = 2
			changeAmt = int(math.Max(float64(changeAmt), 1))
		} else if isKilledNeutral {
			factor = -1
		}
	} else if isKillerNeutral {
		if isKilledGood {
			factor = -1
		} else if isKilledEvil {
			factor = 1
		} else if isKilledNeutral {
			factor = 0
		}
	}

	return factor * changeAmt
}

// expectedDPS estimates average damage per round without randomness.
func expectedDPS(atkChar characters.Character, defChar characters.Character) float64 {

	atkCount := combatAttackCount(atkChar, defChar)

	statDmgBonus := damageBonus(atkChar.Stats.Strength.ValueAdj, defChar.Stats.Strength.ValueAdj)

	attackWeapons := resolveAttackWeapons(atkChar)

	weaponWeight := make([]float64, len(attackWeapons))
	if len(attackWeapons) == 1 {
		weaponWeight[0] = 1.0
	} else {
		dwLevel := atkChar.GetSkillLevel(skills.DualWield)
		alwaysDual := atkChar.Equipment.Weapon.GetSpec().Subtype == items.Claws &&
			atkChar.Equipment.Offhand.GetSpec().Subtype == items.Claws
		switch {
		case alwaysDual || dwLevel >= 3:
			weaponWeight[0] = 1.0
			weaponWeight[1] = 1.0
		case dwLevel == 2:
			weaponWeight[0] = 1.0
			weaponWeight[1] = 0.5
		default:
			weaponWeight[0] = 1.0
			weaponWeight[1] = 0.0
		}
	}

	// hitChance already enforces [ToHitMin, ToHitMax].
	hitPct := float64(hitChance(atkChar.Stats.Speed.ValueAdj, defChar.Stats.Speed.ValueAdj)) / 100.0

	dwLevel := atkChar.GetSkillLevel(skills.DualWield)
	dwPenalty := 0.0
	if len(attackWeapons) > 1 {
		dwPenalty = float64(-dualWieldHitPenalty(dwLevel)) / 100.0
	}

	cfg := configs.GetCombatConfig()
	minHitPct := float64(cfg.ToHitMin) / 100.0

	critPct := float64(critChance(
		atkChar.Stats.Smarts.ValueAdj,
		defChar.Stats.Smarts.ValueAdj,
		false,
		false,
	)) / 100.0

	// A hit that lands is still negated if the defender dodges.
	// Expected damage probability per attack = hitPct * (1 - dodgePct).
	dodgePct := float64(dodgeChance(
		defChar.Stats.Perception.ValueAdj,
		atkChar.Stats.Perception.ValueAdj,
	)) / 100.0

	// Defense reduces damage by an expected fraction of defenseRating/200
	// (average of a uniform roll over [0, defenseRating) divided by 100).
	defenseFraction := float64(defChar.GetDefense()) / 200.0
	if defenseFraction > 0.95 {
		defenseFraction = 0.95
	}

	totalDPS := 0.0

	for roundIdx := 0; roundIdx < atkCount; roundIdx++ {
		for wIdx, weapon := range attackWeapons {
			wWeight := weaponWeight[wIdx]
			if wWeight <= 0 {
				continue
			}

			var attacks, dCount, dSides, dBonus int
			if weapon.ItemId > 0 {
				attacks, dCount, dSides, dBonus, _ = weapon.GetDiceRoll()
			} else {
				attacks, dCount, dSides, dBonus, _ = atkChar.GetDefaultDiceRoll()
			}
			dBonus += statDmgBonus

			avgRoll := float64(dCount) * float64(dSides+1) / 2.0
			avgDmg := avgRoll + float64(dBonus)
			if avgDmg < 0 {
				avgDmg = 0
			}

			critBonusAmt := float64(critDamageBonus(dCount, dSides, dBonus,
				atkChar.Stats.Perception.ValueAdj, defChar.Stats.Perception.ValueAdj))
			critBonus := critBonusAmt * critPct

			effHit := hitPct
			if wIdx > 0 {
				effHit = math.Max(minHitPct, hitPct-dwPenalty)
			}
			// Subtract the expected fraction of hits that get dodged.
			effHit *= (1.0 - dodgePct)

			rawDmg := (avgDmg + critBonus) * effHit
			netDmg := rawDmg * (1.0 - defenseFraction)

			for atkIdx := 0; atkIdx < attacks; atkIdx++ {
				totalDPS += netDmg * wWeight
			}
		}
	}

	return totalDPS
}

// CombatOdds returns the ratio of rounds-for-attacker-to-kill-defender to
// rounds-for-defender-to-kill-attacker. Values above 1.0 favor the attacker.
func CombatOdds(atkChar characters.Character, defChar characters.Character) float64 {
	atkDPS := expectedDPS(atkChar, defChar)
	defDPS := expectedDPS(defChar, atkChar)

	defHP := float64(defChar.Health)
	if defHP < 1 {
		defHP = 1
	}
	atkHP := float64(atkChar.Health)
	if atkHP < 1 {
		atkHP = 1
	}

	if atkDPS < 0.001 {
		return 0
	}

	atkRoundsToKill := defHP / atkDPS

	if defDPS < 0.001 {
		return math.MaxFloat64
	}

	defRoundsToKill := atkHP / defDPS

	return defRoundsToKill / atkRoundsToKill
}
