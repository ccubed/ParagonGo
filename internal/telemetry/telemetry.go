package telemetry

import (
	"encoding/json"
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
	CatCharCreated  = "char_created"
	CatHelpTopic    = "help_topic"
	CatPlayerLogin  = "player_login"
)

// Record holds a single aggregated counter for a unique combination of
// (Date, Category, ItemId, MobId, RoomId, Zone, RaceId, Topic).
type Record struct {
	Date     string `yaml:"date"             json:"date"`
	Category string `yaml:"category"         json:"category"`
	ItemId   int    `yaml:"itemid,omitempty" json:"itemid,omitempty"`
	MobId    int    `yaml:"mobid,omitempty"  json:"mobid,omitempty"`
	RoomId   int    `yaml:"roomid,omitempty" json:"roomid,omitempty"`
	Zone     string `yaml:"zone,omitempty"   json:"zone,omitempty"`
	RaceId   int    `yaml:"raceid,omitempty" json:"raceid,omitempty"`
	Topic    string `yaml:"topic,omitempty"  json:"topic,omitempty"`
	Count    int    `yaml:"count"            json:"count"`
}

var (
	records []Record
	index   = map[string]int{}  // recordKey -> slice index
	dirty   = map[string]bool{} // date -> needs save
	dataDir string              // directory that holds per-date JSON files
)

// Load reads all YYYYMMDD.json files from <dataFilesPath>/telemetry/ and
// rebuilds the in-memory index. Any legacy YYYYMMDD.yaml files found are
// migrated to JSON and then deleted. Safe to call when the directory does
// not exist yet.
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
		if e.IsDir() {
			continue
		}

		name := e.Name()
		path := filepath.Join(dataDir, name)

		switch {
		case strings.HasSuffix(name, ".json"):
			data, err := util.ReadFile(path)
			if err != nil {
				mudlog.Error("telemetry.Load", "file", name, "error", err)
				continue
			}
			var loaded []Record
			if err := json.Unmarshal(data, &loaded); err != nil {
				mudlog.Error("telemetry.Load", "file", name, "error", err)
				continue
			}
			for _, r := range loaded {
				records = append(records, r)
				index[recordKey(r.Date, r.Category, r.Zone, r.ItemId, r.MobId, r.RoomId, r.RaceId, r.Topic)] = len(records) - 1
			}
			fileCount++

		case strings.HasSuffix(name, ".yaml"):
			data, err := util.ReadFile(path)
			if err != nil {
				mudlog.Error("telemetry.Load", "file", name, "error", err)
				continue
			}
			var loaded []Record
			if err := yaml.Unmarshal(data, &loaded); err != nil {
				mudlog.Error("telemetry.Load", "file", name, "error", err)
				continue
			}
			jsonPath := strings.TrimSuffix(path, ".yaml") + ".json"
			if err := writeJSONFile(jsonPath, loaded); err != nil {
				mudlog.Error("telemetry.Load", "action", "migrate", "file", name, "error", err)
				continue
			}
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				mudlog.Error("telemetry.Load", "action", "remove-yaml", "file", name, "error", err)
			}
			mudlog.Info("telemetry.Load", "action", "migrated", "from", name, "to", filepath.Base(jsonPath))
			for _, r := range loaded {
				records = append(records, r)
				index[recordKey(r.Date, r.Category, r.Zone, r.ItemId, r.MobId, r.RoomId, r.RaceId, r.Topic)] = len(records) - 1
			}
			fileCount++
		}
	}

	mudlog.Info("telemetry.Load", "files", fileCount, "records", len(records))
}

// Save writes one JSON file per dirty date to the telemetry directory.
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
		path := filepath.Join(dataDir, date+".json")
		recs := byDate[date]

		if len(recs) == 0 {
			mudlog.Info("telemetry.Save", "action", "delete", "file", path)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				mudlog.Error("telemetry.Save", "action", "delete", "file", path, "error", err)
			}
			continue
		}

		if err := writeJSONFile(path, recs); err != nil {
			mudlog.Error("telemetry.Save", "action", "write", "file", path, "error", err)
			return fmt.Errorf("telemetry.Save write %s: %w", date, err)
		}

		mudlog.Info("telemetry.Save", "action", "wrote", "file", path, "records", len(recs))
	}

	dirty = map[string]bool{}
	return nil
}

// Track increments the counter for the given combination. Zero values for
// numeric fields and empty strings mean "not applicable" for that field.
func Track(category, zone string, itemId, mobId, roomId int) {
	TrackFull(category, zone, itemId, mobId, roomId, 0, "")
}

// TrackFull is like Track but also accepts the optional raceId and topic
// dimensions used by char_created and help_topic records.
func TrackFull(category, zone string, itemId, mobId, roomId, raceId int, topic string) {
	date := time.Now().Format("20060102")
	key := recordKey(date, category, zone, itemId, mobId, roomId, raceId, topic)

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
		RaceId:   raceId,
		Topic:    topic,
		Count:    1,
	})
	index[key] = len(records) - 1
	dirty[date] = true
}

// Clear removes all in-memory records matching the given filters and marks
// the affected dates dirty so Save() will update or remove their files.
// Pass empty/zero values to match any value for that field.
func Clear(category, zone, date, dateTo string, itemId, mobId, roomId, raceId int, topic string) {
	kept := make([]Record, 0, len(records))
	for _, r := range records {
		if matchesFilter(r, category, zone, date, dateTo, itemId, mobId, roomId, raceId, topic) {
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

func writeJSONFile(path string, recs []Record) error {
	data, err := json.Marshal(recs)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return util.WriteFile(path, data, 0644)
}

func rebuildIndex() {
	index = make(map[string]int, len(records))
	for i, r := range records {
		index[recordKey(r.Date, r.Category, r.Zone, r.ItemId, r.MobId, r.RoomId, r.RaceId, r.Topic)] = i
	}
}

func recordKey(date, category, zone string, itemId, mobId, roomId, raceId int, topic string) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|%d|%d|%s", date, category, zone, itemId, mobId, roomId, raceId, topic)
}

// matchesFilter returns true when r matches all non-zero/non-empty filter values.
func matchesFilter(r Record, category, zone, date, dateTo string, itemId, mobId, roomId, raceId int, topic string) bool {
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
	if raceId != 0 && r.RaceId != raceId {
		return false
	}
	if topic != "" && r.Topic != topic {
		return false
	}
	return true
}
