package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

const (
	CatItemDrop     = "item_drop"
	CatItemPickup   = "item_pickup"
	CatMobKill      = "mob_kill"
	CatPlayerDeath  = "player_death"
	CatItemPurchase = "item_purchase"
)

// Record holds a single aggregated counter for a unique combination of
// (Date, Category, ItemId, MobId, RoomId, Zone).
type Record struct {
	Date     string `yaml:"date"             json:"date"`
	Category string `yaml:"category"         json:"category"`
	ItemId   int    `yaml:"itemid,omitempty" json:"itemid,omitempty"`
	MobId    int    `yaml:"mobid,omitempty"  json:"mobid,omitempty"`
	RoomId   int    `yaml:"roomid,omitempty" json:"roomid,omitempty"`
	Zone     string `yaml:"zone,omitempty"   json:"zone,omitempty"`
	Count    int    `yaml:"count"            json:"count"`
}

var (
	records []Record
	index   = map[string]int{}  // recordKey -> slice index
	dirty   = map[string]bool{} // date -> needs save
	dataDir string              // directory that holds per-date YAML files
)

// Load reads all YYYYMMDD.yaml files from <dataFilesPath>/telemetry/ and
// rebuilds the in-memory index. Safe to call when the directory does not
// exist yet.
func Load(dataFilesPath string) {
	dataDir = util.FilePath(dataFilesPath, `/`, `telemetry`)

	mudlog.Info("telemetry.Load", "dir", dataDir)

	records = []Record{}
	index = map[string]int{}
	dirty = map[string]bool{}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			mudlog.Info("telemetry.Load", "status", "directory does not exist yet, will be created on first save")
		} else {
			mudlog.Error("telemetry.Load", "dir", dataDir, "error", err)
		}
		return
	}

	fileCount := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}

		data, err := util.ReadFile(filepath.Join(dataDir, e.Name()))
		if err != nil {
			mudlog.Error("telemetry.Load", "file", e.Name(), "error", err)
			continue
		}

		var loaded []Record
		if err := yaml.Unmarshal(data, &loaded); err != nil {
			mudlog.Error("telemetry.Load", "file", e.Name(), "error", err)
			continue
		}

		for _, r := range loaded {
			records = append(records, r)
			index[recordKey(r.Date, r.Category, r.Zone, r.ItemId, r.MobId, r.RoomId)] = len(records) - 1
		}
		fileCount++
	}

	mudlog.Info("telemetry.Load", "files", fileCount, "records", len(records))
}

// Save writes one YAML file per dirty date to the telemetry directory.
// Dates whose records were all cleared have their file deleted.
// Clean dates are not touched.
func Save() error {
	if dataDir == "" {
		mudlog.Warn("telemetry.Save", "status", "skipped - dataDir not set (Load was never called)")
		return nil
	}

	if len(dirty) == 0 {
		mudlog.Info("telemetry.Save", "status", "skipped - no dirty dates")
		return nil
	}

	mudlog.Info("telemetry.Save", "dir", dataDir, "dirty_dates", len(dirty))

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		mudlog.Error("telemetry.Save", "action", "mkdir", "dir", dataDir, "error", err)
		return fmt.Errorf("telemetry.Save mkdir: %w", err)
	}

	// Group records by date for efficient lookup during save.
	byDate := make(map[string][]Record, len(dirty))
	for _, r := range records {
		if dirty[r.Date] {
			byDate[r.Date] = append(byDate[r.Date], r)
		}
	}

	for date := range dirty {
		path := filepath.Join(dataDir, date+".yaml")
		recs := byDate[date]

		if len(recs) == 0 {
			mudlog.Info("telemetry.Save", "action", "delete", "file", path)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				mudlog.Error("telemetry.Save", "action", "delete", "file", path, "error", err)
			}
			continue
		}

		data, err := yaml.Marshal(recs)
		if err != nil {
			mudlog.Error("telemetry.Save", "action", "marshal", "date", date, "error", err)
			return fmt.Errorf("telemetry.Save marshal %s: %w", date, err)
		}

		if err := util.WriteFile(path, data, 0644); err != nil {
			mudlog.Error("telemetry.Save", "action", "write", "file", path, "error", err)
			return fmt.Errorf("telemetry.Save write %s: %w", date, err)
		}

		mudlog.Info("telemetry.Save", "action", "wrote", "file", path, "records", len(recs))
	}

	dirty = map[string]bool{}
	return nil
}

// Track increments the counter for the given combination. Zero values for
// numeric fields mean "not applicable" for that field.
func Track(category, zone string, itemId, mobId, roomId int) {
	date := time.Now().Format("20060102")
	key := recordKey(date, category, zone, itemId, mobId, roomId)

	if i, ok := index[key]; ok {
		records[i].Count++
		dirty[date] = true
		return
	}

	records = append(records, Record{
		Date:     date,
		Category: category,
		ItemId:   itemId,
		MobId:    mobId,
		RoomId:   roomId,
		Zone:     zone,
		Count:    1,
	})
	index[key] = len(records) - 1
	dirty[date] = true
}

// Clear removes all in-memory records matching the given filters and marks
// the affected dates dirty so Save() will update or remove their files.
// Pass empty/zero values to match any value for that field.
func Clear(category, zone, date, dateTo string, itemId, mobId, roomId int) {
	kept := make([]Record, 0, len(records))
	for _, r := range records {
		if matchesFilter(r, category, zone, date, dateTo, itemId, mobId, roomId) {
			dirty[r.Date] = true
			continue
		}
		kept = append(kept, r)
	}
	records = kept
	rebuildIndex()
}

// Query returns a new QueryBuilder over the current records.
func Query() *QueryBuilder {
	return &QueryBuilder{}
}

// Len returns the total number of records currently in memory.
func Len() int {
	return len(records)
}

func rebuildIndex() {
	index = make(map[string]int, len(records))
	for i, r := range records {
		index[recordKey(r.Date, r.Category, r.Zone, r.ItemId, r.MobId, r.RoomId)] = i
	}
}

func recordKey(date, category, zone string, itemId, mobId, roomId int) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|%d", date, category, zone, itemId, mobId, roomId)
}

// matchesFilter returns true when r matches all non-zero/non-empty filter values.
func matchesFilter(r Record, category, zone, date, dateTo string, itemId, mobId, roomId int) bool {
	if category != "" && r.Category != category {
		return false
	}
	if zone != "" && r.Zone != zone {
		return false
	}
	if date != "" && r.Date < date {
		return false
	}
	if dateTo != "" && r.Date > dateTo {
		return false
	}
	if itemId != 0 && r.ItemId != itemId {
		return false
	}
	if mobId != 0 && r.MobId != mobId {
		return false
	}
	if roomId != 0 && r.RoomId != roomId {
		return false
	}
	return true
}
