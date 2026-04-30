package combat

/*
func loadSimTestData(t *testing.T) {
	t.Helper()
	loadTestData(t)
	races.LoadDataFiles()
	buffs.LoadDataFiles()
	mobs.LoadDataFiles()
}

func TestSimulateCombat_Basic(t *testing.T) {
	loadSimTestData(t)

	allMobs := mobs.GetAllMobInfo()
	if len(allMobs) < 2 {
		t.Skip("need at least 2 mob templates to test simulation")
	}

	mobA := allMobs[0]
	mobB := allMobs[1]

	result, err := SimulateCombat(mobA.MobId, mobB.MobId, 0, 0, 200)
	if err != nil {
		t.Fatalf("SimulateCombat: %v", err)
	}

	if result.Rounds < 1 {
		t.Errorf("expected at least 1 round, got %d", result.Rounds)
	}

	if result.WinnerSide < 0 || result.WinnerSide > 2 {
		t.Errorf("unexpected WinnerSide: %d", result.WinnerSide)
	}

	if result.WinnerSide != 0 && result.Winner == "" {
		t.Error("non-draw result should have a winner name")
	}

	if result.DamageByA < 0 || result.DamageByB < 0 {
		t.Errorf("damage should not be negative: A=%d B=%d", result.DamageByA, result.DamageByB)
	}
}

func TestSimulateCombat_LevelOverride(t *testing.T) {
	loadSimTestData(t)

	allMobs := mobs.GetAllMobInfo()
	if len(allMobs) < 1 {
		t.Skip("need at least 1 mob template")
	}

	mobId := allMobs[0].MobId

	result, err := SimulateCombat(mobId, mobId, 5, 20, 200)
	if err != nil {
		t.Fatalf("SimulateCombat: %v", err)
	}

	if result.LevelA != 5 || result.LevelB != 20 {
		t.Errorf("expected levels 5 vs 20, got %d vs %d", result.LevelA, result.LevelB)
	}

	if result.Rounds < 1 {
		t.Errorf("expected at least 1 round, got %d", result.Rounds)
	}
}

func TestSimulateCombat_InvalidMob(t *testing.T) {
	loadSimTestData(t)

	_, err := SimulateCombat(999999, 999998, 0, 0, 100)
	if err == nil {
		t.Error("expected error for invalid mob IDs")
	}
}

func TestSimulateCombat_Output(t *testing.T) {
	loadSimTestData(t)

	allMobs := mobs.GetAllMobInfo()
	if len(allMobs) < 2 {
		t.Skip("need at least 2 mob templates")
	}

	result, err := SimulateCombat(allMobs[0].MobId, allMobs[1].MobId, 0, 0, 200)
	if err != nil {
		t.Fatalf("SimulateCombat: %v", err)
	}

	fmt.Println(result.String())
}
*/
