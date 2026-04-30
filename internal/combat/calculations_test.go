package combat

import (
	"fmt"
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/races"
)

func TestAlignmentChange(t *testing.T) {
	tests := []struct {
		killerAlignment int8
		killedAlignment int8
		expectedChange  int
	}{

		{0, 0, 0},
		{0, 5, 0},
		{5, 0, 0},

		{0, 15, 0},
		{0, -15, 0},

		{15, -15, 0},
		{-15, 15, 0},

		{15, 25, 0},
		{25, 15, 0},

		{-20, -25, 2},
		{-25, -20, 2},

		{50, -10, -1},
		{-50, 10, -1},

		{50, -50, 2},
		{-50, 50, -2},

		{90, 0, -2},
		{-90, 0, -2},

		{100, 20, -4},
		{-100, -20, 4},

		{90, -90, 4},
		{-90, 90, -4},
	}

	for _, test := range tests {
		desc := fmt.Sprintf(`%s kills %s`, characters.AlignmentToString(test.killerAlignment), characters.AlignmentToString(test.killedAlignment))
		delta := int(math.Abs(math.Max(float64(test.killerAlignment), float64(test.killedAlignment))-math.Min(float64(test.killerAlignment), float64(test.killedAlignment))) * 0.5)
		result := AlignmentChange(test.killerAlignment, test.killedAlignment)
		if result != test.expectedChange {
			t.Errorf("%s [Delta: %d]: AlignmentChange(%d, %d) = %d; want %d",
				desc, delta, test.killerAlignment, test.killedAlignment, result, test.expectedChange)
		}
	}
}

// TestStatDelta verifies the clamped fractional delta helper.
func TestStatDelta(t *testing.T) {
	tests := []struct {
		atk, def int
		want     float64
	}{
		{100, 0, 1.0}, // full advantage
		{0, 100, 0.0}, // no advantage
		{50, 0, 0.5},  // half
		{0, 0, 0.0},   // equal
		{200, 0, 1.0}, // clamped at 1
		{50, 50, 0.0}, // equal stats
		{75, 25, 0.5}, // 50 delta
	}
	for _, tt := range tests {
		got := statDelta(tt.atk, tt.def)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("statDelta(%d, %d) = %g; want %g", tt.atk, tt.def, got, tt.want)
		}
	}
}

// TestDamageBonus verifies damage bonus uses Strength delta and config bounds.
// Default config: min=0, max=10.
func TestDamageBonus(t *testing.T) {
	tests := []struct {
		atkStr, defStr int
		wantMin        int
		wantMax        int
	}{
		{0, 0, 0, 0},     // equal -> 0
		{100, 0, 10, 10}, // full delta -> max (10)
		{50, 0, 5, 5},    // half delta -> 5
		{0, 100, 0, 0},   // no advantage -> min (0)
	}
	for _, tt := range tests {
		got := damageBonus(tt.atkStr, tt.defStr)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("damageBonus(%d, %d) = %d; want [%d, %d]", tt.atkStr, tt.defStr, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestHitChance verifies hit chance uses Speed delta and config bounds.
// Default config: min=25, max=100.
func TestHitChance(t *testing.T) {
	tests := []struct {
		atkSpd, defSpd int
		wantMin        int
		wantMax        int
	}{
		{0, 0, 50, 50},     // equal -> min (25)
		{100, 0, 150, 150}, // full advantage -> max (100)
		{50, 0, 100, 100},  // half delta -> floor(0.5*100)=50
		{0, 100, 50, 50},   // no advantage -> min (25)
	}
	for _, tt := range tests {
		got := hitChance(tt.atkSpd, tt.defSpd)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("hitChance(%d, %d) = %d; want [%d, %d]", tt.atkSpd, tt.defSpd, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestExtraAttackCount verifies extra attacks use Speed delta and config bounds.
// Default config: min=0, max=3.
func TestExtraAttackCount(t *testing.T) {
	tests := []struct {
		atkSpd, defSpd int
		wantMin        int
		wantMax        int
	}{
		{0, 0, 0, 0},   // equal -> 0
		{100, 0, 3, 3}, // full delta -> 3
		{50, 0, 1, 2},  // half delta -> floor(0.5*3)=1
		{0, 100, 0, 0}, // no advantage -> 0
	}
	for _, tt := range tests {
		got := extraAttackCount(tt.atkSpd, tt.defSpd)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("extraAttackCount(%d, %d) = %d; want [%d, %d]", tt.atkSpd, tt.defSpd, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestWeaponlessAttackCount verifies total weaponless attack count.
func TestWeaponlessAttackCount(t *testing.T) {
	tests := []struct {
		atkSpd, defSpd, mod int
		wantMin             int
		wantMax             int
	}{
		{0, 0, 0, 1, 1},   // 1 base + 0 extra
		{100, 0, 0, 4, 4}, // 1 base + 3 extra
		{0, 0, -5, 1, 1},  // floored to 1
		{100, 0, 1, 5, 5}, // 1 + 3 + 1 mod
	}
	for _, tt := range tests {
		got := weaponlessAttackCount(tt.atkSpd, tt.defSpd, tt.mod)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("weaponlessAttackCount(%d, %d, %d) = %d; want [%d, %d]",
				tt.atkSpd, tt.defSpd, tt.mod, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestCritChance verifies crit chance uses Smarts delta and config bounds.
// Default config: min=5, max=30.
func TestCritChance(t *testing.T) {
	tests := []struct {
		atkSmarts, defSmarts int
		hasAccuracy          bool
		targetHasBlink       bool
		wantMin              int
		wantMax              int
	}{
		{0, 0, false, false, 5, 5},     // equal -> min (5)
		{100, 0, false, false, 30, 30}, // full delta -> max (30)
		{50, 0, false, false, 15, 15},  // half delta -> floor(0.5*30)=15
		{0, 100, false, false, 5, 5},   // no advantage -> min (5)
		// accuracy doubles, capped at 100
		{100, 0, true, false, 60, 60},
		// blink halves
		{100, 0, false, true, 15, 15},
		// both: 30*2/2 = 30
		{100, 0, true, true, 30, 30},
		// min enforced after blink
		{0, 0, false, true, 5, 5},
	}
	for _, tt := range tests {
		got := critChance(tt.atkSmarts, tt.defSmarts, tt.hasAccuracy, tt.targetHasBlink)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("critChance(%d, %d, %v, %v) = %d; want [%d, %d]",
				tt.atkSmarts, tt.defSmarts, tt.hasAccuracy, tt.targetHasBlink, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestCritMultiplier verifies crit multiplier uses Perception delta.
// Default config: min=1.5, max=3.0.
func TestCritMultiplier(t *testing.T) {
	tests := []struct {
		atkPerc, defPerc int
		wantMin          float64
		wantMax          float64
	}{
		{0, 0, 1.5, 1.5},   // equal -> min (1.5)
		{100, 0, 3.0, 3.0}, // full delta -> max (3.0)
		{50, 0, 1.5, 1.5},  // half delta -> 0.5*3=1.5 (equals min)
		{0, 100, 1.5, 1.5}, // no advantage -> min (1.5)
	}
	for _, tt := range tests {
		got := critMultiplier(tt.atkPerc, tt.defPerc)
		if got < tt.wantMin-1e-9 || got > tt.wantMax+1e-9 {
			t.Errorf("critMultiplier(%d, %d) = %g; want [%g, %g]",
				tt.atkPerc, tt.defPerc, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestCritDamageBonus verifies the crit bonus scales with the multiplier.
func TestCritDamageBonus(t *testing.T) {
	tests := []struct {
		dCount, dSides, dBonus int
		atkPerc, defPerc       int
		wantMin                int
		wantMax                int
	}{
		// base=12, mult=1.5 -> bonus = floor(12*(1.5-1)) = floor(6) = 6
		{2, 6, 0, 0, 0, 6, 6},
		// base=12, mult=3.0 -> bonus = floor(12*(3.0-1)) = floor(24) = 24
		{2, 6, 0, 100, 0, 24, 24},
		// base=0, any -> 0
		{0, 0, 0, 100, 0, 0, 0},
		// negative base clamped to 0
		{1, 4, -10, 0, 0, 0, 0},
	}
	for _, tt := range tests {
		got := critDamageBonus(tt.dCount, tt.dSides, tt.dBonus, tt.atkPerc, tt.defPerc)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("critDamageBonus(%d,%d,%d,%d,%d) = %d; want [%d, %d]",
				tt.dCount, tt.dSides, tt.dBonus, tt.atkPerc, tt.defPerc, got, tt.wantMin, tt.wantMax)
		}
	}
}

// TestDodgeChance verifies dodge chance uses Perception delta and config bounds.
// Default config: min=5, max=30.
func TestDodgeChance(t *testing.T) {
	tests := []struct {
		defPerc, atkPerc int
		wantMin          int
		wantMax          int
	}{
		{0, 0, 5, 5},     // equal -> min (5)
		{100, 0, 30, 30}, // full advantage -> max (30)
		{0, 100, 5, 5},   // no advantage -> min (5)
		{50, 0, 15, 15},  // half delta -> floor(0.5*30)=15
	}
	for _, tt := range tests {
		got := dodgeChance(tt.defPerc, tt.atkPerc)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("dodgeChance(%d, %d) = %d; want [%d, %d]", tt.defPerc, tt.atkPerc, got, tt.wantMin, tt.wantMax)
		}
	}
}

func TestDualWieldHitPenalty(t *testing.T) {
	tests := []struct {
		dwLevel int
		want    int
	}{
		{0, -35},
		{1, -35},
		{2, -35},
		{3, -35},
		{4, -25},
		{5, -25},
	}
	for _, tt := range tests {
		got := dualWieldHitPenalty(tt.dwLevel)
		if got != tt.want {
			t.Errorf("dualWieldHitPenalty(%d) = %d; want %d", tt.dwLevel, got, tt.want)
		}
	}
}

func TestDualWieldActiveWeaponCount(t *testing.T) {
	// Deterministic cases
	tests := []struct {
		dwLevel   int
		bothClaws bool
		want      int
	}{
		{0, false, 1},
		{1, false, 1},
		{3, false, 2},
		{4, false, 2},
		{0, true, 2},
		{1, true, 2},
	}
	for _, tt := range tests {
		got := dualWieldActiveWeaponCount(tt.dwLevel, tt.bothClaws)
		if got != tt.want {
			t.Errorf("dualWieldActiveWeaponCount(%d, %v) = %d; want %d", tt.dwLevel, tt.bothClaws, got, tt.want)
		}
	}

	// Probabilistic case: dwLevel == 2, bothClaws == false should return 1 or 2
	saw1, saw2 := false, false
	for i := 0; i < 200; i++ {
		v := dualWieldActiveWeaponCount(2, false)
		if v == 1 {
			saw1 = true
		} else if v == 2 {
			saw2 = true
		} else {
			t.Errorf("dualWieldActiveWeaponCount(2, false) returned unexpected value %d", v)
		}
		if saw1 && saw2 {
			break
		}
	}
	if !saw1 || !saw2 {
		t.Errorf("dualWieldActiveWeaponCount(2, false) did not produce both 1 and 2 over 200 iterations")
	}
}

func TestApplyDefenseReduction(t *testing.T) {
	// Zero defense: no reduction ever
	for i := 0; i < 50; i++ {
		final, red := applyDefenseReduction(100, 0)
		if final != 100 || red != 0 {
			t.Errorf("applyDefenseReduction(100, 0) = (%d, %d); want (100, 0)", final, red)
		}
	}

	// Non-zero defense: final + reduction == original damage
	for i := 0; i < 100; i++ {
		final, red := applyDefenseReduction(100, 50)
		if final+red != 100 {
			t.Errorf("applyDefenseReduction(100, 50): final(%d) + reduction(%d) != 100", final, red)
		}
		if final < 0 || red < 0 {
			t.Errorf("applyDefenseReduction(100, 50): negative value final=%d red=%d", final, red)
		}
	}
}

func TestDamagePercentOfMax(t *testing.T) {
	tests := []struct {
		damage, dCount, dSides, dBonus int
		want                           int
	}{
		{0, 2, 6, 0, 0},    // 0% of max
		{12, 2, 6, 0, 100}, // max damage = 12, 100%
		{6, 2, 6, 0, 50},   // half max
		{1, 0, 0, 0, 100},  // maxDmg clamped to 1, 100%
		{5, 2, 6, 0, 42},   // ceil(5/12*100) = 42
	}
	for _, tt := range tests {
		got := damagePercentOfMax(tt.damage, tt.dCount, tt.dSides, tt.dBonus)
		if got != tt.want {
			t.Errorf("damagePercentOfMax(%d, %d, %d, %d) = %d; want %d",
				tt.damage, tt.dCount, tt.dSides, tt.dBonus, got, tt.want)
		}
	}
}

func TestTameSizeModifier(t *testing.T) {
	tests := []struct {
		size races.Size
		want int
	}{
		{races.Large, -25},
		{races.Small, 0},
		{races.Medium, -10},
		{races.Size("unknown"), -10},
	}
	for _, tt := range tests {
		got := tameSizeModifier(tt.size)
		if got != tt.want {
			t.Errorf("tameSizeModifier(%q) = %d; want %d", tt.size, got, tt.want)
		}
	}
}

func TestTameHealthBonus(t *testing.T) {
	correctTests := []struct {
		currentHP, maxHP int
		want             float64
	}{
		{100, 100, 0},
		{50, 100, 25},
		{1, 100, 49},
		{0, 100, 50},
	}
	for _, tt := range correctTests {
		got := tameHealthBonus(tt.currentHP, tt.maxHP)
		if got != tt.want {
			t.Errorf("tameHealthBonus(%d, %d) = %g; want %g", tt.currentHP, tt.maxHP, got, tt.want)
		}
	}
}

func TestChanceToTame(t *testing.T) {
	tests := []struct {
		proficiency   int
		levelDiff     int
		currentHP     int
		maxHP         int
		tamerSize     races.Size
		targetIsAggro bool
		wantMin       int
		wantMax       int
	}{
		{100, 10, 100, 100, races.Medium, false, 100, 110},
		{1, -25, 100, 100, races.Medium, false, -34, -33},
		{50, 0, 100, 100, races.Medium, true, 20, 21},
		{50, 0, 100, 100, races.Large, false, 25, 25},
		{50, 0, 100, 100, races.Small, false, 50, 50},
		{-10, 0, 100, 100, races.Medium, false, -9, -9},
		{200, 0, 100, 100, races.Medium, false, 90, 90},
		{50, 100, 100, 100, races.Medium, false, 65, 65},
		{50, -100, 100, 100, races.Medium, false, 15, 15},
	}
	for _, tt := range tests {
		got := chanceToTame(tt.proficiency, tt.levelDiff, tt.currentHP, tt.maxHP, tt.tamerSize, tt.targetIsAggro)
		if got < tt.wantMin || got > tt.wantMax {
			t.Errorf("chanceToTame(%d, %d, %d, %d, %q, %v) = %d; want [%d, %d]",
				tt.proficiency, tt.levelDiff, tt.currentHP, tt.maxHP, tt.tamerSize, tt.targetIsAggro,
				got, tt.wantMin, tt.wantMax)
		}
	}
}
