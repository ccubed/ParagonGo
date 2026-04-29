package combat

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func loadTestData(t *testing.T) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir to repo root: %v", err)
	}
	mudlog.SetupLogger(nil, "", "", false)
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	items.LoadDataFiles()
}

func TestRankWeapons_Compiles(t *testing.T) {
	loadTestData(t)

	byDPS, byAdjDPS, byMaxDmg := RankWeapons()

	if len(byDPS) == 0 {
		t.Fatal("RankWeapons returned no weapons")
	}
	if len(byDPS) != len(byAdjDPS) || len(byDPS) != len(byMaxDmg) {
		t.Fatalf("slice length mismatch: byDPS=%d byAdjDPS=%d byMaxDmg=%d",
			len(byDPS), len(byAdjDPS), len(byMaxDmg))
	}

	// DPS slice must be non-increasing.
	for i := 1; i < len(byDPS); i++ {
		if byDPS[i].DPS > byDPS[i-1].DPS {
			t.Errorf("byDPS not sorted at index %d: %.3f > %.3f", i, byDPS[i].DPS, byDPS[i-1].DPS)
		}
	}

	// AdjDPS slice must be non-increasing.
	for i := 1; i < len(byAdjDPS); i++ {
		if byAdjDPS[i].AdjDPS > byAdjDPS[i-1].AdjDPS {
			t.Errorf("byAdjDPS not sorted at index %d: %.3f > %.3f", i, byAdjDPS[i].AdjDPS, byAdjDPS[i-1].AdjDPS)
		}
	}

	// MaxDmg slice must be non-increasing.
	for i := 1; i < len(byMaxDmg); i++ {
		if byMaxDmg[i].MaxDmg > byMaxDmg[i-1].MaxDmg {
			t.Errorf("byMaxDmg not sorted at index %d: %d > %d", i, byMaxDmg[i].MaxDmg, byMaxDmg[i-1].MaxDmg)
		}
	}
}

func TestFormatWeaponRankings_Output(t *testing.T) {
	loadTestData(t)
	fmt.Println(FormatWeaponRankings())
}
