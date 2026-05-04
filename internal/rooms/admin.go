package rooms

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type RoomSummary struct {
	RoomId        int    `json:"RoomId"`
	Zone          string `json:"Zone"`
	Title         string `json:"Title"`
	Biome         string `json:"Biome"`
	ExitCount     int    `json:"ExitCount"`
	SpawnCount    int    `json:"SpawnCount"`
	IsBank        bool   `json:"IsBank,omitempty"`
	Pvp           bool   `json:"Pvp,omitempty"`
	HasScript     bool   `json:"HasScript,omitempty"`
	TrainingCount int    `json:"TrainingCount,omitempty"`
}

type PaginatedRooms struct {
	Rooms   []RoomSummary `json:"rooms"`
	Total   int           `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"perPage"`
}

type ZoneSummary struct {
	Name         string   `json:"Name"`
	RoomId       int      `json:"RoomId"`
	RoomCount    int      `json:"RoomCount"`
	DefaultBiome string   `json:"DefaultBiome"`
	MusicFile    string   `json:"MusicFile,omitempty"`
	AutoScaleMin int      `json:"AutoScaleMin"`
	AutoScaleMax int      `json:"AutoScaleMax"`
	IdleMessages []string `json:"IdleMessages,omitempty"`
}

func GetAllZoneSummaries() []ZoneSummary {
	result := make([]ZoneSummary, 0, len(roomManager.zones))
	for _, zc := range roomManager.zones {
		result = append(result, ZoneSummary{
			Name:         zc.Name,
			RoomId:       zc.RoomId,
			RoomCount:    len(zc.RoomIds),
			DefaultBiome: zc.DefaultBiome,
			MusicFile:    zc.MusicFile,
			AutoScaleMin: zc.MobAutoScale.Minimum,
			AutoScaleMax: zc.MobAutoScale.Maximum,
			IdleMessages: zc.IdleMessages,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func GetPaginatedRoomSummaries(zone string, search string, page int, perPage int) PaginatedRooms {
	all := perPage == -1
	if page < 1 {
		page = 1
	}
	if !all {
		if perPage < 1 {
			perPage = 50
		}
		if perPage > 1000 {
			perPage = 1000
		}
	}

	search = strings.ToLower(strings.TrimSpace(search))

	var matchingIds []int

	if zone != "" {
		zoneInfo, ok := roomManager.zones[zone]
		if !ok {
			return PaginatedRooms{Rooms: []RoomSummary{}, Page: page, PerPage: perPage}
		}
		matchingIds = make([]int, 0, len(zoneInfo.RoomIds))
		for roomId := range zoneInfo.RoomIds {
			matchingIds = append(matchingIds, roomId)
		}
	} else {
		matchingIds = make([]int, 0, len(roomManager.roomIdToFileCache))
		for roomId := range roomManager.roomIdToFileCache {
			matchingIds = append(matchingIds, roomId)
		}
	}

	if search != "" {
		filtered := matchingIds[:0]
		for _, id := range matchingIds {
			if strings.Contains(fmt.Sprintf("%d", id), search) {
				filtered = append(filtered, id)
				continue
			}
			if summary, ok := roomManager.roomSummaries[id]; ok {
				if strings.Contains(strings.ToLower(summary.Title), search) {
					filtered = append(filtered, id)
				}
			}
		}
		matchingIds = filtered
	}

	sort.Ints(matchingIds)

	total := len(matchingIds)
	var pageIds []int
	if all {
		pageIds = matchingIds
		perPage = total
	} else {
		start := (page - 1) * perPage
		if start > total {
			start = total
		}
		end := start + perPage
		if end > total {
			end = total
		}
		pageIds = matchingIds[start:end]
	}
	rooms := make([]RoomSummary, 0, len(pageIds))

	for _, roomId := range pageIds {
		r := LoadRoomTemplate(roomId)
		if r == nil {
			if summary, ok := roomManager.roomSummaries[roomId]; ok {
				rooms = append(rooms, RoomSummary{
					RoomId: roomId,
					Zone:   summary.Zone,
					Title:  summary.Title,
					Biome:  summary.Biome,
				})
			}
			continue
		}
		rooms = append(rooms, RoomSummary{
			RoomId:        r.RoomId,
			Zone:          r.Zone,
			Title:         r.Title,
			Biome:         r.Biome,
			ExitCount:     len(r.Exits),
			SpawnCount:    len(r.SpawnInfo),
			IsBank:        r.IsBank,
			Pvp:           r.Pvp,
			HasScript:     r.GetScript() != "",
			TrainingCount: len(r.SkillTraining),
		})
	}

	return PaginatedRooms{
		Rooms:   rooms,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}
}

func GetRoomForAdmin(roomId int) *Room {
	return LoadRoomTemplate(roomId)
}

func SaveRoomForAdmin(room Room) error {
	if room.RoomId < 1 {
		return fmt.Errorf("invalid room id: %d", room.RoomId)
	}

	if err := room.Validate(); err != nil {
		return err
	}

	if err := SaveRoomTemplate(room); err != nil {
		return err
	}

	roomManager.roomSummaries[room.RoomId] = RoomSummaryInfo{
		Title: room.Title,
		Zone:  room.Zone,
		Biome: room.Biome,
	}

	return nil
}

func CreateRoomForAdmin(zone string) (int, error) {
	if _, ok := roomManager.zones[zone]; !ok {
		return 0, fmt.Errorf("zone does not exist: %s", zone)
	}

	newRoom := NewRoom(zone)
	if err := newRoom.Validate(); err != nil {
		return 0, err
	}

	addRoomToMemory(newRoom)

	if err := SaveRoomTemplate(*newRoom); err != nil {
		return 0, err
	}

	roomManager.roomSummaries[newRoom.RoomId] = RoomSummaryInfo{
		Title: newRoom.Title,
		Zone:  newRoom.Zone,
		Biome: newRoom.Biome,
	}

	return newRoom.RoomId, nil
}

func DeleteRoomForAdmin(roomId int) error {
	if roomId < 1 {
		return fmt.Errorf("invalid room id: %d", roomId)
	}

	summary, ok := roomManager.roomSummaries[roomId]
	if !ok {
		return fmt.Errorf("room not found: %d", roomId)
	}

	if zoneInfo, ok := roomManager.zones[summary.Zone]; ok {
		if zoneInfo.RoomId == roomId {
			return fmt.Errorf("cannot delete the root room of zone %q (room %d)", summary.Zone, roomId)
		}
	}

	if r, ok := roomManager.rooms[roomId]; ok {
		if len(r.players) > 0 {
			return fmt.Errorf("cannot delete room %d: players are present", roomId)
		}
	}

	basePath := configs.GetFilePathsConfig().DataFiles.String()

	filePath, ok := roomManager.roomIdToFileCache[roomId]
	if ok {
		templatePath := util.FilePath(basePath, `/rooms/`, filePath)
		if err := os.Remove(templatePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing room template: %w", err)
		}

		instancePath := util.FilePath(basePath, `/rooms.instances/`, filePath)
		os.Remove(instancePath)
	}

	ClearRoomCache(roomId)
	delete(roomManager.roomIdToFileCache, roomId)
	delete(roomManager.roomSummaries, roomId)

	if zoneInfo, ok := roomManager.zones[summary.Zone]; ok {
		delete(zoneInfo.RoomIds, roomId)
	}

	return nil
}

func SaveZoneConfigForAdmin(name string, cfg *ZoneConfig) error {
	existing, ok := roomManager.zones[name]
	if !ok {
		return fmt.Errorf("zone does not exist: %s", name)
	}

	cfg.Name = existing.Name
	cfg.RoomId = existing.RoomId
	cfg.RoomIds = existing.RoomIds

	if err := cfg.Validate(); err != nil {
		return err
	}

	roomsBasePath := configs.GetFilePathsConfig().DataFiles.String() + `/rooms`

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*ZoneConfig](roomsBasePath, cfg, saveModes...); err != nil {
		return err
	}

	roomManager.zones[name] = cfg
	return nil
}

// SaveRoomScript writes (or overwrites) the JavaScript file for a room. If
// content is empty the script file is deleted instead.
func SaveRoomScript(roomId int, content string) error {
	r := LoadRoomTemplate(roomId)
	if r == nil {
		return fmt.Errorf("room %d not found", roomId)
	}

	scriptPath := r.GetScriptPath()

	if content == "" {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing room script: %w", err)
		}
		return nil
	}

	return os.WriteFile(scriptPath, []byte(content), 0644)
}

type MapperRoomData struct {
	RoomId         int                       `json:"RoomId"`
	Zone           string                    `json:"Zone"`
	Title          string                    `json:"Title"`
	MapX           int                       `json:"MapX"`
	MapY           int                       `json:"MapY"`
	MapZ           int                       `json:"MapZ"`
	HasCoordinates bool                      `json:"HasCoordinates"`
	MapSymbol      string                    `json:"MapSymbol"`
	MapLegend      string                    `json:"MapLegend"`
	Biome          string                    `json:"Biome"`
	Exits          map[string]MapperExitData `json:"Exits"`
}

type MapperExitData struct {
	RoomId       int    `json:"RoomId"`
	Secret       bool   `json:"Secret,omitempty"`
	MapDirection string `json:"MapDirection,omitempty"`
	HasLock      bool   `json:"HasLock,omitempty"`
}

func GetMapperRooms(zoneName string) ([]MapperRoomData, error) {
	zoneInfo, ok := roomManager.zones[zoneName]
	if !ok {
		return nil, fmt.Errorf("zone does not exist: %s", zoneName)
	}

	roomIds := make([]int, 0, len(zoneInfo.RoomIds))
	for id := range zoneInfo.RoomIds {
		roomIds = append(roomIds, id)
	}
	sort.Ints(roomIds)

	result := make([]MapperRoomData, 0, len(roomIds))
	for _, id := range roomIds {
		r := LoadRoomTemplate(id)
		if r == nil {
			continue
		}

		exits := make(map[string]MapperExitData, len(r.Exits))
		for dir, ex := range r.Exits {
			exits[dir] = MapperExitData{
				RoomId:       ex.RoomId,
				Secret:       ex.Secret,
				MapDirection: ex.MapDirection,
				HasLock:      ex.HasLock(),
			}
		}

		result = append(result, MapperRoomData{
			RoomId:         r.RoomId,
			Zone:           r.Zone,
			Title:          r.Title,
			MapX:           r.MapX,
			MapY:           r.MapY,
			MapZ:           r.MapZ,
			HasCoordinates: r.HasCoordinates,
			MapSymbol:      r.MapSymbol,
			MapLegend:      r.MapLegend,
			Biome:          r.Biome,
			Exits:          exits,
		})
	}

	return result, nil
}

func GetAllMapperRooms() []MapperRoomData {
	roomIds := make([]int, 0, len(roomManager.roomIdToFileCache))
	for id := range roomManager.roomIdToFileCache {
		roomIds = append(roomIds, id)
	}
	sort.Ints(roomIds)

	result := make([]MapperRoomData, 0, len(roomIds))
	for _, id := range roomIds {
		r := LoadRoomTemplate(id)
		if r == nil {
			continue
		}

		exits := make(map[string]MapperExitData, len(r.Exits))
		for dir, ex := range r.Exits {
			exits[dir] = MapperExitData{
				RoomId:       ex.RoomId,
				Secret:       ex.Secret,
				MapDirection: ex.MapDirection,
				HasLock:      ex.HasLock(),
			}
		}

		result = append(result, MapperRoomData{
			RoomId:         r.RoomId,
			Zone:           r.Zone,
			Title:          r.Title,
			MapX:           r.MapX,
			MapY:           r.MapY,
			MapZ:           r.MapZ,
			HasCoordinates: r.HasCoordinates,
			MapSymbol:      r.MapSymbol,
			MapLegend:      r.MapLegend,
			Biome:          r.Biome,
			Exits:          exits,
		})
	}

	return result
}

func DeleteZoneForAdmin(zoneName string) error {
	zoneInfo, ok := roomManager.zones[zoneName]
	if !ok {
		return fmt.Errorf("zone does not exist: %s", zoneName)
	}

	for roomId := range zoneInfo.RoomIds {
		if r, ok := roomManager.rooms[roomId]; ok {
			if len(r.players) > 0 {
				return fmt.Errorf("cannot delete zone %q: room %d has players", zoneName, roomId)
			}
		}
	}

	basePath := configs.GetFilePathsConfig().DataFiles.String()
	zoneFolder := ZoneToFolder(zoneName)

	templateDir := util.FilePath(basePath, `/rooms/`, zoneFolder)
	instanceDir := util.FilePath(basePath, `/rooms.instances/`, zoneFolder)

	for roomId := range zoneInfo.RoomIds {
		delete(roomManager.rooms, roomId)
		delete(roomManager.roomsWithUsers, roomId)
		delete(roomManager.roomsWithMobs, roomId)
		delete(roomManager.roomIdToFileCache, roomId)
		delete(roomManager.roomSummaries, roomId)
	}

	delete(roomManager.zones, zoneName)

	os.RemoveAll(templateDir)
	os.RemoveAll(instanceDir)

	return nil
}
