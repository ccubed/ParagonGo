package telemetry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

func resetState() {
	records = []Record{}
	index = map[string]int{}
	dirty = map[string]bool{}
	dataDir = ""
}

func TestTrack_NewRecord(t *testing.T) {
	resetState()

	Track(CatMobKill, "frostfang", 0, 7, 100)

	require.Len(t, records, 1)
	assert.Equal(t, CatMobKill, records[0].Category)
	assert.Equal(t, "frostfang", records[0].Zone)
	assert.Equal(t, 7, records[0].MobId)
	assert.Equal(t, 100, records[0].RoomId)
	assert.Equal(t, 1, records[0].Count)
}

func TestTrack_Increment(t *testing.T) {
	resetState()

	Track(CatMobKill, "frostfang", 0, 7, 100)
	Track(CatMobKill, "frostfang", 0, 7, 100)
	Track(CatMobKill, "frostfang", 0, 7, 100)

	require.Len(t, records, 1)
	assert.Equal(t, 3, records[0].Count)
}

func TestTrack_DifferentDimensions(t *testing.T) {
	resetState()

	Track(CatItemDrop, "zone1", 42, 7, 100)
	Track(CatItemDrop, "zone1", 42, 8, 100)
	Track(CatItemDrop, "zone2", 42, 7, 200)

	assert.Len(t, records, 3)
}

func TestTrack_MarksDirty(t *testing.T) {
	resetState()

	Track(CatMobKill, "z", 0, 1, 10)

	assert.Len(t, dirty, 1)
	for date := range dirty {
		assert.Len(t, date, 8) // YYYYMMDD
	}
}

func TestQuery_FilterByCategory(t *testing.T) {
	resetState()

	Track(CatMobKill, "zone1", 0, 1, 10)
	Track(CatItemDrop, "zone1", 5, 1, 10)
	Track(CatMobKill, "zone1", 0, 2, 11)

	results := Query().Category(CatMobKill).Results()
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, CatMobKill, r.Category)
	}
}

func TestQuery_FilterByItemId(t *testing.T) {
	resetState()

	Track(CatItemDrop, "zone1", 42, 7, 100)
	Track(CatItemDrop, "zone1", 43, 7, 100)
	Track(CatItemDrop, "zone1", 42, 8, 101)

	results := Query().Category(CatItemDrop).ItemId(42).Results()
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, 42, r.ItemId)
	}
}

func TestQuery_FilterByMobId(t *testing.T) {
	resetState()

	Track(CatItemDrop, "zone1", 42, 7, 100)
	Track(CatItemDrop, "zone1", 43, 7, 100)
	Track(CatItemDrop, "zone1", 42, 8, 101)

	results := Query().Category(CatItemDrop).MobId(7).Results()
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, 7, r.MobId)
	}
}

func TestQuery_FilterByZone(t *testing.T) {
	resetState()

	Track(CatMobKill, "frostfang", 0, 1, 10)
	Track(CatMobKill, "frostfang", 0, 2, 11)
	Track(CatMobKill, "dungeon", 0, 3, 20)

	results := Query().Zone("frostfang").Results()
	assert.Len(t, results, 2)
}

func TestQuery_FilterByDateRange(t *testing.T) {
	resetState()

	records = []Record{
		{Date: "20260101", Category: CatMobKill, MobId: 1, Count: 5},
		{Date: "20260201", Category: CatMobKill, MobId: 2, Count: 3},
		{Date: "20260301", Category: CatMobKill, MobId: 3, Count: 7},
	}
	rebuildIndex()

	results := Query().DateFrom("20260201").DateTo("20260301").Results()
	assert.Len(t, results, 2)
}

func TestQuery_FilterByExactDate(t *testing.T) {
	resetState()

	records = []Record{
		{Date: "20260101", Category: CatMobKill, MobId: 1, Count: 5},
		{Date: "20260201", Category: CatMobKill, MobId: 2, Count: 3},
	}
	rebuildIndex()

	results := Query().Date("20260101").Results()
	require.Len(t, results, 1)
	assert.Equal(t, 1, results[0].MobId)
}

func TestQuery_SortDesc(t *testing.T) {
	resetState()

	Track(CatMobKill, "z", 0, 1, 10)
	Track(CatMobKill, "z", 0, 1, 10)
	Track(CatMobKill, "z", 0, 2, 11)

	results := Query().Category(CatMobKill).SortDesc().Results()
	require.Len(t, results, 2)
	assert.True(t, results[0].Count >= results[1].Count)
}

func TestQuery_SortAsc(t *testing.T) {
	resetState()

	Track(CatMobKill, "z", 0, 1, 10)
	Track(CatMobKill, "z", 0, 1, 10)
	Track(CatMobKill, "z", 0, 2, 11)

	results := Query().Category(CatMobKill).SortAsc().Results()
	require.Len(t, results, 2)
	assert.True(t, results[0].Count <= results[1].Count)
}

func TestQuery_Total(t *testing.T) {
	resetState()

	Track(CatMobKill, "z", 0, 1, 10)
	Track(CatMobKill, "z", 0, 1, 10)
	Track(CatMobKill, "z", 0, 2, 11)

	total := Query().Category(CatMobKill).Total()
	assert.Equal(t, 3, total)
}

func TestClear(t *testing.T) {
	resetState()

	Track(CatMobKill, "frostfang", 0, 1, 10)
	Track(CatItemDrop, "frostfang", 5, 1, 10)
	Track(CatMobKill, "dungeon", 0, 2, 20)

	Clear(CatMobKill, "frostfang", "", "", 0, 0, 0)

	results := Query().Category(CatMobKill).Results()
	assert.Len(t, results, 1)
	assert.Equal(t, "dungeon", results[0].Zone)

	allResults := Query().Results()
	assert.Len(t, allResults, 2)
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	resetState()

	base := t.TempDir()
	dataDir = filepath.Join(base, "telemetry")

	// Inject two records with known dates so we can assert per-file layout.
	records = []Record{
		{Date: "20260101", Category: CatMobKill, MobId: 7, Zone: "zone1", Count: 2},
		{Date: "20260102", Category: CatItemDrop, ItemId: 42, Zone: "zone2", Count: 1},
	}
	rebuildIndex()
	dirty["20260101"] = true
	dirty["20260102"] = true

	require.NoError(t, Save())

	// Verify two separate files were written.
	assert.FileExists(t, filepath.Join(dataDir, "20260101.yaml"))
	assert.FileExists(t, filepath.Join(dataDir, "20260102.yaml"))

	// Reload from directory.
	resetState()
	Load(base)

	results := Query().Results()
	require.Len(t, results, 2)

	kills := Query().Category(CatMobKill).Results()
	require.Len(t, kills, 1)
	assert.Equal(t, 2, kills[0].Count)

	drops := Query().Category(CatItemDrop).Results()
	require.Len(t, drops, 1)
	assert.Equal(t, 42, drops[0].ItemId)
}

func TestSave_DeletesFileWhenDateCleared(t *testing.T) {
	resetState()

	base := t.TempDir()
	dataDir = filepath.Join(base, "telemetry")

	records = []Record{
		{Date: "20260101", Category: CatMobKill, MobId: 1, Count: 3},
		{Date: "20260102", Category: CatMobKill, MobId: 2, Count: 1},
	}
	rebuildIndex()
	dirty["20260101"] = true
	dirty["20260102"] = true
	require.NoError(t, Save())

	// Clear only the first date.
	Clear("", "", "20260101", "20260101", 0, 0, 0)
	require.NoError(t, Save())

	assert.NoFileExists(t, filepath.Join(dataDir, "20260101.yaml"))
	assert.FileExists(t, filepath.Join(dataDir, "20260102.yaml"))
}

func TestLoad_EmptyDirectory(t *testing.T) {
	resetState()

	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "telemetry"), 0755))
	Load(base)

	assert.Empty(t, records)
	assert.Empty(t, index)
}

func TestQuery_GroupByMobId(t *testing.T) {
	resetState()

	// Same mob killed in two different rooms on two different dates.
	records = []Record{
		{Date: "20260101", Category: CatMobKill, MobId: 7, Zone: "frostfang", RoomId: 100, Count: 3},
		{Date: "20260102", Category: CatMobKill, MobId: 7, Zone: "frostfang", RoomId: 101, Count: 5},
		{Date: "20260101", Category: CatMobKill, MobId: 9, Zone: "dungeon", RoomId: 200, Count: 2},
	}
	rebuildIndex()

	results := Query().Category(CatMobKill).GroupBy(GroupByMobId).SortDesc().Results()

	require.Len(t, results, 2)
	// Mob 7 should be first with count 8.
	assert.Equal(t, 7, results[0].MobId)
	assert.Equal(t, 8, results[0].Count)
	// Date, Zone, RoomId are zeroed out on rolled-up records.
	assert.Empty(t, results[0].Date)
	assert.Empty(t, results[0].Zone)
	assert.Equal(t, 0, results[0].RoomId)
	// Mob 9 second.
	assert.Equal(t, 9, results[1].MobId)
	assert.Equal(t, 2, results[1].Count)
}

func TestQuery_GroupByZone(t *testing.T) {
	resetState()

	records = []Record{
		{Date: "20260101", Category: CatMobKill, MobId: 1, Zone: "frostfang", Count: 4},
		{Date: "20260102", Category: CatMobKill, MobId: 2, Zone: "frostfang", Count: 6},
		{Date: "20260101", Category: CatMobKill, MobId: 3, Zone: "dungeon", Count: 1},
	}
	rebuildIndex()

	results := Query().Category(CatMobKill).GroupBy(GroupByZone).SortDesc().Results()

	require.Len(t, results, 2)
	assert.Equal(t, "frostfang", results[0].Zone)
	assert.Equal(t, 10, results[0].Count)
	assert.Equal(t, "dungeon", results[1].Zone)
	assert.Equal(t, 1, results[1].Count)
}

func TestLoad_MissingDirectory(t *testing.T) {
	resetState()

	Load(t.TempDir()) // telemetry subdir does not exist

	assert.Empty(t, records)
	assert.Empty(t, index)
}

func TestOnMobDeath_SkipsSuicide(t *testing.T) {
	resetState()

	// No KilledByUsers means the mob died with no player involvement (suicide).
	onMobDeath(events.MobDeath{
		MobId:         7,
		RoomId:        100,
		KilledByUsers: []int{},
	})

	assert.Empty(t, records)
}

func TestOnMobDeath_TracksPlayerKill(t *testing.T) {
	resetState()

	onMobDeath(events.MobDeath{
		MobId:         7,
		RoomId:        100,
		KilledByUsers: []int{42},
	})

	require.Len(t, records, 1)
	assert.Equal(t, CatMobKill, records[0].Category)
	assert.Equal(t, 7, records[0].MobId)
}

func TestTrackFull_CharCreated_ByRace(t *testing.T) {
	resetState()

	TrackFull(CatCharCreated, "", 0, 0, 0, 3, "")
	TrackFull(CatCharCreated, "", 0, 0, 0, 3, "")
	TrackFull(CatCharCreated, "", 0, 0, 0, 5, "")

	require.Len(t, records, 2)

	results := Query().Category(CatCharCreated).GroupBy(GroupByRaceId).SortDesc().Results()
	require.Len(t, results, 2)
	assert.Equal(t, 3, results[0].RaceId)
	assert.Equal(t, 2, results[0].Count)
	assert.Equal(t, 5, results[1].RaceId)
	assert.Equal(t, 1, results[1].Count)
}

func TestTrackFull_CharCreated_FilterByRaceId(t *testing.T) {
	resetState()

	TrackFull(CatCharCreated, "", 0, 0, 0, 3, "")
	TrackFull(CatCharCreated, "", 0, 0, 0, 5, "")

	total := Query().Category(CatCharCreated).RaceId(3).Total()
	assert.Equal(t, 1, total)
}

func TestTrackFull_HelpTopic(t *testing.T) {
	resetState()

	TrackFull(CatHelpTopic, "", 0, 0, 0, 0, "attack")
	TrackFull(CatHelpTopic, "", 0, 0, 0, 0, "attack")
	TrackFull(CatHelpTopic, "", 0, 0, 0, 0, "spells")

	require.Len(t, records, 2)

	results := Query().Category(CatHelpTopic).GroupBy(GroupByTopic).SortDesc().Results()
	require.Len(t, results, 2)
	assert.Equal(t, "attack", results[0].Topic)
	assert.Equal(t, 2, results[0].Count)
	assert.Equal(t, "spells", results[1].Topic)
	assert.Equal(t, 1, results[1].Count)
}

func TestTrackFull_HelpTopic_FilterByTopic(t *testing.T) {
	resetState()

	TrackFull(CatHelpTopic, "", 0, 0, 0, 0, "attack")
	TrackFull(CatHelpTopic, "", 0, 0, 0, 0, "spells")

	total := Query().Category(CatHelpTopic).Topic("attack").Total()
	assert.Equal(t, 1, total)
}
