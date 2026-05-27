package rooms

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
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
			HasScript:     r.HasScript(),
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

	roomsBasePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/rooms`)

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

// GetRoomInstanceRaw returns the raw YAML bytes of the instance file for
// roomId. Returns nil, nil when the file does not exist.
func GetRoomInstanceRaw(roomId int) ([]byte, error) {
	filePath, ok := roomManager.roomIdToFileCache[roomId]
	if !ok {
		return nil, fmt.Errorf("room %d not found", roomId)
	}
	instancePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/rooms.instances/`, filePath)
	data, err := util.ReadFile(instancePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// SaveRoomInstanceRaw writes raw YAML bytes to the instance file for roomId.
// The room template must exist. Passing empty or nil content deletes the file.
func SaveRoomInstanceRaw(roomId int, content []byte) error {
	if LoadRoomTemplate(roomId) == nil {
		return fmt.Errorf("room %d not found", roomId)
	}
	if len(content) == 0 {
		return DeleteRoomInstance(roomId)
	}
	filePath, ok := roomManager.roomIdToFileCache[roomId]
	if !ok {
		return fmt.Errorf("room %d not found", roomId)
	}
	instancePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/rooms.instances/`, filePath)
	dir := strings.TrimSuffix(instancePath, fmt.Sprintf("%d.yaml", roomId))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating instance directory: %w", err)
	}
	return util.WriteFile(instancePath, content, 0644)
}

// DeleteRoomInstance removes the instance file for roomId. Returns nil when
// the file does not exist.
func DeleteRoomInstance(roomId int) error {
	filePath, ok := roomManager.roomIdToFileCache[roomId]
	if !ok {
		return fmt.Errorf("room %d not found", roomId)
	}
	instancePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/rooms.instances/`, filePath)
	if err := os.Remove(instancePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing instance file: %w", err)
	}
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

	return util.WriteFile(scriptPath, []byte(content), 0644)
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
	Biome          string                    `json:"Biome,omitempty"`
	Tags           []string                  `json:"Tags,omitempty"`
	HasScript      bool                      `json:"HasScript,omitempty"`
	HasMobSpawn    bool                      `json:"HasMobSpawn,omitempty"`
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

		biome := r.Biome
		if biome == "" {
			biome = zoneInfo.DefaultBiome
		}

		hasMobSpawn := false
		for _, si := range r.SpawnInfo {
			if si.MobId > 0 {
				hasMobSpawn = true
				break
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
			Biome:          biome,
			Tags:           r.Tags,
			HasScript:      r.HasScript(),
			HasMobSpawn:    hasMobSpawn,
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

		biome := r.Biome
		if biome == "" {
			if zi, ok := roomManager.zones[r.Zone]; ok {
				biome = zi.DefaultBiome
			}
		}

		hasMobSpawn2 := false
		for _, si := range r.SpawnInfo {
			if si.MobId > 0 {
				hasMobSpawn2 = true
				break
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
			Biome:          biome,
			Tags:           r.Tags,
			HasScript:      r.HasScript(),
			HasMobSpawn:    hasMobSpawn2,
			Exits:          exits,
		})
	}

	return result
}

// RenameZoneForAdmin renames a zone: it updates all on-disk files (rooms,
// mobs, conversations, zone-config) and all in-memory state (room manager,
// live mob instances, online user records). No players may be present in the
// zone when this is called.
func RenameZoneForAdmin(oldName, newName string) error {
	zoneInfo, ok := roomManager.zones[oldName]
	if !ok {
		return fmt.Errorf("zone does not exist: %s", oldName)
	}

	newName = strings.TrimSpace(newName)
	if err := ValidateZoneName(newName); err != nil {
		return err
	}

	if _, exists := roomManager.zones[newName]; exists {
		return fmt.Errorf("zone already exists: %s", newName)
	}

	basePath := configs.GetFilePathsConfig().DataFiles.String()
	oldFolder := ZoneToFolder(oldName)
	newFolder := ZoneToFolder(newName)

	oldRoomsDir := util.FilePath(basePath, `/rooms/`, oldFolder)
	newRoomsDir := util.FilePath(basePath, `/rooms/`, newFolder)
	oldInstancesDir := util.FilePath(basePath, `/rooms.instances/`, oldFolder)
	newInstancesDir := util.FilePath(basePath, `/rooms.instances/`, newFolder)
	oldMobsDir := util.FilePath(basePath, `/mobs/`, oldFolder)
	newMobsDir := util.FilePath(basePath, `/mobs/`, newFolder)
	oldConvsDir := util.FilePath(basePath, `/conversations/`, ZoneNameSanitize(oldName))
	newConvsDir := util.FilePath(basePath, `/conversations/`, ZoneNameSanitize(newName))

	// Rename rooms template folder.
	if err := os.Rename(oldRoomsDir, newRoomsDir); err != nil {
		return fmt.Errorf("renaming rooms folder: %w", err)
	}

	// Rename rooms instances folder (may not exist).
	if err := os.Rename(oldInstancesDir, newInstancesDir); err != nil && !os.IsNotExist(err) {
		// Roll back rooms folder rename.
		os.Rename(newRoomsDir, oldRoomsDir)
		return fmt.Errorf("renaming rooms.instances folder: %w", err)
	}

	// Rename mobs folder (may not exist).
	if err := os.Rename(oldMobsDir, newMobsDir); err != nil && !os.IsNotExist(err) {
		os.Rename(newRoomsDir, oldRoomsDir)
		os.Rename(newInstancesDir, oldInstancesDir)
		return fmt.Errorf("renaming mobs folder: %w", err)
	}

	// Rename conversations folder (may not exist).
	if err := os.Rename(oldConvsDir, newConvsDir); err != nil && !os.IsNotExist(err) {
		os.Rename(newRoomsDir, oldRoomsDir)
		os.Rename(newInstancesDir, oldInstancesDir)
		os.Rename(newMobsDir, oldMobsDir)
		return fmt.Errorf("renaming conversations folder: %w", err)
	}

	// Update zone-config.yaml: name field.
	zoneInfo.Name = newName
	roomsBasePath := util.FilePath(basePath + `/rooms`)
	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}
	if err := fileloader.SaveFlatFile[*ZoneConfig](roomsBasePath, zoneInfo, saveModes...); err != nil {
		return fmt.Errorf("saving zone config: %w", err)
	}

	// Rewrite the file cache and in-memory Zone fields for every room in the
	// zone before touching disk.
	oldFolderPrefix := ZoneToFolder(oldName)
	newFolderPrefix := ZoneToFolder(newName)
	for roomId := range zoneInfo.RoomIds {
		if cached, ok := roomManager.roomIdToFileCache[roomId]; ok {
			roomManager.roomIdToFileCache[roomId] = strings.Replace(cached, oldFolderPrefix, newFolderPrefix, 1)
		}
		if r, ok := roomManager.rooms[roomId]; ok {
			r.Zone = newName
		}
		if summary, ok := roomManager.roomSummaries[roomId]; ok {
			summary.Zone = newName
			roomManager.roomSummaries[roomId] = summary
		}
	}

	// Patch the zone: field in every room YAML on disk by rewriting the raw
	// bytes. We cannot use LoadRoomTemplate here because LoadFlatFile validates
	// that the file path ends in Filepath(), which is derived from the zone
	// field stored inside the YAML — but those YAMLs still contain the old
	// zone name until we overwrite them.
	oldZoneYAML := fmt.Sprintf("zone: %s", oldName)
	newZoneYAML := fmt.Sprintf("zone: %s", newName)
	for roomId := range zoneInfo.RoomIds {
		filePath, ok := roomManager.roomIdToFileCache[roomId]
		if !ok {
			continue
		}
		fullPath := util.FilePath(basePath, `/rooms/`, filePath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		patched := strings.Replace(string(data), oldZoneYAML, newZoneYAML, 1)
		os.WriteFile(fullPath, []byte(patched), 0644)
	}

	// Update every mob spec on disk and in memory.
	for _, spec := range mobs.GetAllMobSpecs() {
		if spec.Zone != oldName {
			continue
		}
		spec.Zone = newName
		mobs.SaveMobSpec(&spec)
	}

	// Update in-memory room manager zone index.
	delete(roomManager.zones, oldName)
	roomManager.zones[newName] = zoneInfo

	// Move coordinate index.
	if coordIdx, ok := roomManager.coordinateIndex[oldName]; ok {
		roomManager.coordinateIndex[newName] = coordIdx
		delete(roomManager.coordinateIndex, oldName)
	}

	// Update live mob instances.
	for _, m := range mobs.GetAllMobInstances() {
		if m.Character.Zone == oldName {
			m.Character.Zone = newName
		}
	}

	// Update online users.
	for _, u := range users.GetAllActiveUsers() {
		if u.Character.Zone == oldName {
			u.Character.Zone = newName
		}
		if u.Character.ZonesVisited != nil {
			if bs, ok := u.Character.ZonesVisited[oldName]; ok {
				u.Character.ZonesVisited[newName] = bs
				delete(u.Character.ZonesVisited, oldName)
			}
		}
	}

	return nil
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
