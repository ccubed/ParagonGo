package web

import (
	"net/http"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type MapperZoneResponse struct {
	ZoneName     string                 `json:"ZoneName"`
	RootRoomId   int                    `json:"RootRoomId"`
	DefaultBiome string                 `json:"DefaultBiome"`
	Rooms        []rooms.MapperRoomData `json:"Rooms"`
}

type MapperAllRoomsResponse struct {
	Zones []rooms.ZoneSummary    `json:"Zones"`
	Rooms []rooms.MapperRoomData `json:"Rooms"`
}

// GET /admin/api/v1/mapper/rooms
func apiV1GetMapperAllRooms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[MapperAllRoomsResponse]{
		Success: true,
		Data: MapperAllRoomsResponse{
			Zones: rooms.GetAllZoneSummaries(),
			Rooms: rooms.GetAllMapperRooms(),
		},
	})
}

// GET /admin/api/v1/mapper/zone/{zonename}
func apiV1GetMapperZone(w http.ResponseWriter, r *http.Request) {
	zoneName := r.PathValue("zonename")

	zc := rooms.GetZoneConfig(zoneName)
	if zc == nil {
		writeAPIError(w, http.StatusNotFound, "zone not found: "+zoneName)
		return
	}

	mapperRooms, err := rooms.GetMapperRooms(zoneName)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[MapperZoneResponse]{
		Success: true,
		Data: MapperZoneResponse{
			ZoneName:     zc.Name,
			RootRoomId:   zc.RoomId,
			DefaultBiome: zc.DefaultBiome,
			Rooms:        mapperRooms,
		},
	})
}
