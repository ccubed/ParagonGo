package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// ---------------------------------------------------------------------------
// Zones
// ---------------------------------------------------------------------------

// GET /admin/api/v1/zones
func apiV1GetZones(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[[]rooms.ZoneSummary]{
		Success: true,
		Data:    rooms.GetAllZoneSummaries(),
	})
}

// POST /admin/api/v1/zones
func apiV1CreateZone(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"Name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if body.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if err := rooms.ValidateZoneName(body.Name); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	roomId, err := rooms.CreateZone(body.Name)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"Name":   body.Name,
			"RoomId": roomId,
		},
	})
}

// PATCH /admin/api/v1/zones/{zonename}
func apiV1PatchZone(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("zonename")

	existing := rooms.GetZoneConfig(zoneName)
	if existing == nil {
		writeAPIError(w, http.StatusNotFound, "zone not found: "+zoneName)
		return
	}

	updated := *existing
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := rooms.SaveZoneConfigForAdmin(zoneName, &updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*rooms.ZoneConfig]{
		Success: true,
		Data:    rooms.GetZoneConfig(zoneName),
	})
}

// DELETE /admin/api/v1/zones/{zonename}
func apiV1DeleteZone(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("zonename")

	if err := rooms.DeleteZoneForAdmin(zoneName); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// POST /admin/api/v1/zones/{zonename}/rename
func apiV1RenameZone(w http.ResponseWriter, r *http.Request) {
	oldName := r.PathValue("zonename")

	var body struct {
		NewName string `json:"NewName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if body.NewName == "" {
		writeAPIError(w, http.StatusBadRequest, "NewName is required")
		return
	}

	if err := rooms.ValidateZoneName(body.NewName); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := rooms.RenameZoneForAdmin(oldName, body.NewName); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"OldName": oldName,
			"NewName": body.NewName,
		},
	})
}

// ---------------------------------------------------------------------------
// Rooms
// ---------------------------------------------------------------------------

// GET /admin/api/v1/rooms/biomes
func apiV1GetBiomes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[[]rooms.BiomeInfo]{
		Success: true,
		Data:    rooms.GetAllBiomes(),
	})
}

// GET /admin/api/v1/rooms
func apiV1GetRooms(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	search := r.URL.Query().Get("search")

	if r.URL.Query().Get("all") == "true" {
		result := rooms.GetPaginatedRoomSummaries(zone, search, 1, -1)
		writeJSON(w, http.StatusOK, APIResponse[rooms.PaginatedRooms]{
			Success: true,
			Data:    result,
		})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("perPage"))
	if perPage < 1 {
		perPage = 50
	}

	result := rooms.GetPaginatedRoomSummaries(zone, search, page, perPage)

	writeJSON(w, http.StatusOK, APIResponse[rooms.PaginatedRooms]{
		Success: true,
		Data:    result,
	})
}

// POST /admin/api/v1/rooms
func apiV1CreateRoom(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Zone string `json:"Zone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if body.Zone == "" {
		writeAPIError(w, http.StatusBadRequest, "Zone is required")
		return
	}

	roomId, err := rooms.CreateRoomForAdmin(body.Zone)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	created := rooms.GetRoomForAdmin(roomId)
	writeJSON(w, http.StatusCreated, APIResponse[*rooms.Room]{
		Success: true,
		Data:    created,
	})
}

// GET /admin/api/v1/rooms/{roomId}
func apiV1GetRoom(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	room := rooms.GetRoomForAdmin(roomId)
	if room == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*rooms.Room]{
		Success: true,
		Data:    room,
	})
}

// PATCH /admin/api/v1/rooms/{roomId}
func apiV1PatchRoom(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	existing := rooms.GetRoomForAdmin(roomId)
	if existing == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	updated := *existing
	// Nil out map fields that must be fully replaced (not merged) by the JSON decode.
	// Go's json.Decoder merges into existing maps, so without this, removing a key
	// (e.g. deleting an exit) would leave the old key intact.
	updated.Exits = nil
	updated.Nouns = nil
	updated.SkillTraining = nil
	updated.Containers = nil
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	updated.RoomId = roomId

	if updated.Title == "" {
		writeAPIError(w, http.StatusBadRequest, "title cannot be empty")
		return
	}
	if updated.GetDescription() == "" {
		writeAPIError(w, http.StatusBadRequest, "description cannot be empty")
		return
	}

	if err := rooms.SaveRoomForAdmin(updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	saved := rooms.GetRoomForAdmin(roomId)
	writeJSON(w, http.StatusOK, APIResponse[*rooms.Room]{
		Success: true,
		Data:    saved,
	})
}

// DELETE /admin/api/v1/rooms/{roomId}
func apiV1DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	if err := rooms.DeleteRoomForAdmin(roomId); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/rooms/{roomId}/script
func apiV1GetRoomScript(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	room := rooms.GetRoomForAdmin(roomId)
	if room == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": room.GetScript(), "lang": scriptLangString(room.GetScriptPath())},
	})
}

// PUT /admin/api/v1/rooms/{roomId}/script
func apiV1PutRoomScript(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	if rooms.GetRoomForAdmin(roomId) == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	var body struct {
		Script string `json:"script"`
		Lang   string `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := rooms.SaveRoomScript(roomId, body.Script, body.Lang); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scripting.InvalidateRoomVM(roomId)

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/rooms/{roomId}/instance
func apiV1GetRoomInstance(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	if rooms.GetRoomForAdmin(roomId) == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	data, err := rooms.GetRoomInstanceRaw(roomId)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"instance": string(data)},
	})
}

// PUT /admin/api/v1/rooms/{roomId}/instance
func apiV1PutRoomInstance(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	if rooms.GetRoomForAdmin(roomId) == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	var body struct {
		Instance string `json:"instance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := rooms.SaveRoomInstanceRaw(roomId, []byte(body.Instance)); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// DELETE /admin/api/v1/rooms/{roomId}/instance
func apiV1DeleteRoomInstance(w http.ResponseWriter, r *http.Request) {
	roomId, err := strconv.Atoi(r.PathValue("roomId"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "roomId must be an integer")
		return
	}

	if rooms.GetRoomForAdmin(roomId) == nil {
		writeAPIError(w, http.StatusNotFound, "room not found: "+strconv.Itoa(roomId))
		return
	}

	if err := rooms.DeleteRoomInstance(roomId); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
