package combat

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
)

type SimResult struct {
	Winner           string
	WinnerSide       int // 0=draw, 1=A won, 2=B won
	Rounds           int
	DamageByA        int
	DamageByB        int
	HealthRemainingA int
	HealthRemainingB int
	NameA            string
	NameB            string
	LevelA           int
	LevelB           int
	Log              []string
}

func (r SimResult) String() string {
	result := fmt.Sprintf("%s (Lv%d) vs %s (Lv%d)\n", r.NameA, r.LevelA, r.NameB, r.LevelB)
	for _, line := range r.Log {
		result += line + "\n"
	}
	result += fmt.Sprintf("Winner: %s after %d rounds\n", r.Winner, r.Rounds)
	result += fmt.Sprintf("Damage dealt: %s=%d, %s=%d\n", r.NameA, r.DamageByA, r.NameB, r.DamageByB)
	result += fmt.Sprintf("Health remaining: %s=%d, %s=%d\n", r.NameA, r.HealthRemainingA, r.NameB, r.HealthRemainingB)
	return result
}

// newSimMob creates a fully initialized mob from a template without registering
// it in the global instance map. If forceLevel > 0, the mob's level is overridden.
func newSimMob(mobId mobs.MobId, forceLevel int) (*mobs.Mob, error) {
	mob := mobs.GetMobSpec(mobId)
	if mob == nil {
		return nil, fmt.Errorf("mob %d not found", mobId)
	}

	if forceLevel > 0 {
		mob.Character.Level = forceLevel
	}

	mob.Character.PlayerDamage = make(map[int]int)
	mob.Character.StatPoints = 0
	{
		cfgProg := configs.GetProgressionConfig()
		for lvl := 1; lvl <= mob.Character.Level; lvl++ {
			if int(cfgProg.StatPointsEveryNLevels) <= 1 || lvl%int(cfgProg.StatPointsEveryNLevels) == 0 {
				mob.Character.StatPoints += int(cfgProg.StatPointsPerLevel)
			}
		}
	}
	mob.Character.Level--
	mob.Character.Experience = mob.Character.XPTNL()
	mob.Character.Level++

	mob.Character.AutoTrain()
	mob.Character.Health = mob.Character.HealthMax.Value
	mob.Character.Mana = mob.Character.ManaMax.Value

	mob.Character.SetPermaBuffs(mob.BuffIds)
	mob.Character.Buffs = buffs.New()

	for idx := range mob.Character.Items {
		mob.Character.Items[idx].Validate()
	}

	if mob.Character.Alignment == 0 {
		if raceInfo := races.GetRace(mob.Character.GetRaceId()); raceInfo != nil {
			if raceInfo.DefaultAlignment != 0 {
				mob.Character.Alignment = raceInfo.DefaultAlignment
			}
		}
	}

	for _, slot := range characters.AllSlots() {
		mob.Character.Equipment.Get(slot).Validate()
	}

	mob.Validate()
	mob.Character.Validate(true)

	return mob, nil
}

// SimulateCombat runs an instant fight between two mobs identified by their
// template IDs. levelA/levelB override the template level when > 0.
// maxRounds caps the fight length (defaults to 100 if <= 0).
func SimulateCombat(mobIdA, mobIdB mobs.MobId, levelA, levelB int, maxRounds int) (SimResult, error) {
	if maxRounds <= 0 {
		maxRounds = 100
	}

	mobA, err := newSimMob(mobIdA, levelA)
	if err != nil {
		return SimResult{}, fmt.Errorf("combatant A: %w", err)
	}
	mobB, err := newSimMob(mobIdB, levelB)
	if err != nil {
		return SimResult{}, fmt.Errorf("combatant B: %w", err)
	}

	charA := &mobA.Character
	charB := &mobB.Character

	charA.SetAggro(0, mobB.InstanceId, characters.DefaultAttack)
	charB.SetAggro(0, mobA.InstanceId, characters.DefaultAttack)

	charA.CancelBuffsWithFlag(buffs.CancelIfCombat)
	charB.CancelBuffsWithFlag(buffs.CancelIfCombat)

	result := SimResult{
		NameA:  charA.Name,
		NameB:  charB.Name,
		LevelA: charA.Level,
		LevelB: charB.Level,
	}

	for round := 1; round <= maxRounds; round++ {
		roundDmgA, roundDmgB := 0, 0

		// A attacks B
		atkResult := calculateCombat(*charA, *charB, Mob, Mob)
		charB.ApplyHealthChange(atkResult.DamageToTarget * -1)
		charA.ApplyHealthChange(atkResult.DamageToSource * -1)
		result.DamageByA += atkResult.DamageToTarget
		roundDmgA = atkResult.DamageToTarget
		applySimBuffs(charA, atkResult.BuffSource)
		applySimBuffs(charB, atkResult.BuffTarget)

		if charB.Health <= 0 {
			result.Winner = charA.Name
			result.WinnerSide = 1
			result.Rounds = round
			result.HealthRemainingA = charA.Health
			result.HealthRemainingB = charB.Health
			result.Log = append(result.Log, fmt.Sprintf(
				"Round %d: %s deals %d → %s falls (%d hp)",
				round, charA.Name, roundDmgA, charB.Name, charB.Health))
			return result, nil
		}

		// B attacks A
		defResult := calculateCombat(*charB, *charA, Mob, Mob)
		charA.ApplyHealthChange(defResult.DamageToTarget * -1)
		charB.ApplyHealthChange(defResult.DamageToSource * -1)
		result.DamageByB += defResult.DamageToTarget
		roundDmgB = defResult.DamageToTarget
		applySimBuffs(charB, defResult.BuffSource)
		applySimBuffs(charA, defResult.BuffTarget)

		if charA.Health <= 0 {
			result.Winner = charB.Name
			result.WinnerSide = 2
			result.Rounds = round
			result.HealthRemainingA = charA.Health
			result.HealthRemainingB = charB.Health
			result.Log = append(result.Log, fmt.Sprintf(
				"Round %d: %s deals %d, %s deals %d → %s falls (%d hp)",
				round, charA.Name, roundDmgA, charB.Name, roundDmgB, charA.Name, charA.Health))
			return result, nil
		}

		result.Log = append(result.Log, fmt.Sprintf(
			"Round %d: %s deals %d (%s hp: %d), %s deals %d (%s hp: %d)",
			round,
			charA.Name, roundDmgA, charB.Name, charB.Health,
			charB.Name, roundDmgB, charA.Name, charA.Health))
	}

	result.Winner = "draw"
	result.WinnerSide = 0
	result.Rounds = maxRounds
	result.HealthRemainingA = charA.Health
	result.HealthRemainingB = charB.Health
	return result, nil
}

func applySimBuffs(char *characters.Character, buffIds []int) {
	for _, buffId := range buffIds {
		char.AddBuff(buffId, false)
	}
}
