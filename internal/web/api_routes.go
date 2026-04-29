package web

import (
	"net/http"
)

func registerAdminAPIRoutes(mux *http.ServeMux) {
	// Tags
	mux.HandleFunc("GET /admin/api/v1/tags", RunWithMUDLocked(
		doBasicAuth(apiV1GetTags),
	))

	// Stats
	mux.HandleFunc("GET /admin/api/v1/stats/memory", RunWithMUDLocked(
		doBasicAuth(apiV1GetStatsMemory),
	))

	// Connections
	mux.HandleFunc("GET /admin/api/v1/connections", RunWithMUDLocked(
		doBasicAuth(apiV1GetConnections),
	))

	// Config
	mux.HandleFunc("GET /admin/api/v1/config", RunWithMUDLocked(
		doBasicAuth(apiV1GetConfig),
	))
	mux.HandleFunc("PATCH /admin/api/v1/config", RunWithMUDLocked(
		doBasicAuth(RunInTestMode(apiV1PatchConfig)),
	))

	// StatMods
	mux.HandleFunc("GET /admin/api/v1/statmods", RunWithMUDLocked(
		doBasicAuth(apiV1GetStatMods),
	))

	// Items — static sub-routes must be registered before the wildcard {itemId}
	// pattern so the Go 1.22 ServeMux prefers the more specific match.
	mux.HandleFunc("GET /admin/api/v1/items/types", RunWithMUDLocked(
		doBasicAuth(apiV1GetItemTypes),
	))
	mux.HandleFunc("GET /admin/api/v1/items/attack-messages", RunWithMUDLocked(
		doBasicAuth(apiV1GetItemAttackMessages),
	))
	mux.HandleFunc("GET /admin/api/v1/items/ranks", RunWithMUDLocked(
		doBasicAuth(apiV1GetItemRanks),
	))
	mux.HandleFunc("PUT /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}", RunWithMUDLocked(
		doBasicAuth(apiV1PutItemAttackMessage),
	))
	mux.HandleFunc("DELETE /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}/{index}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteItemAttackMessage),
	))
	mux.HandleFunc("GET /admin/api/v1/items", RunWithMUDLocked(
		doBasicAuth(apiV1GetItems),
	))
	mux.HandleFunc("POST /admin/api/v1/items", RunWithMUDLocked(
		doBasicAuth(apiV1CreateItem),
	))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1GetItemScript),
	))
	mux.HandleFunc("PUT /admin/api/v1/items/{itemId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1PutItemScript),
	))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}", RunWithMUDLocked(
		doBasicAuth(apiV1GetItem),
	))
	mux.HandleFunc("PATCH /admin/api/v1/items/{itemId}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchItem),
	))
	mux.HandleFunc("DELETE /admin/api/v1/items/{itemId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteItem),
	))

	// Buffs
	mux.HandleFunc("GET /admin/api/v1/buffs", RunWithMUDLocked(
		doBasicAuth(apiV1GetBuffs),
	))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1GetBuffScript),
	))
	mux.HandleFunc("PUT /admin/api/v1/buffs/{buffId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1PutBuffScript),
	))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}", RunWithMUDLocked(
		doBasicAuth(apiV1GetBuff),
	))
	mux.HandleFunc("PATCH /admin/api/v1/buffs/{buffId}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchBuff),
	))
	mux.HandleFunc("DELETE /admin/api/v1/buffs/{buffId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteBuff),
	))

	// Quests
	mux.HandleFunc("GET /admin/api/v1/quests", RunWithMUDLocked(
		doBasicAuth(apiV1GetQuests),
	))
	mux.HandleFunc("PATCH /admin/api/v1/quests", RunWithMUDLocked(
		doBasicAuth(apiV1PatchQuest),
	))
	mux.HandleFunc("DELETE /admin/api/v1/quests/{questId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteQuest),
	))

	// Users — static sub-routes must be registered before any future wildcard
	// {userId} pattern.
	mux.HandleFunc("GET /admin/api/v1/users/search", RunWithMUDLocked(
		doBasicAuth(apiV1SearchUsers),
	))
	mux.HandleFunc("POST /admin/api/v1/users", RunWithMUDLocked(
		doBasicAuth(apiV1CreateUser),
	))
	mux.HandleFunc("GET /admin/api/v1/users/{userid}", RunWithMUDLocked(
		doBasicAuth(apiV1GetUser),
	))
	mux.HandleFunc("PATCH /admin/api/v1/users/{userid}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchUser),
	))

	// Color Aliases
	mux.HandleFunc("GET /admin/api/v1/color-aliases", RunWithMUDLocked(
		doBasicAuth(apiV1GetColorAliases),
	))
	mux.HandleFunc("PATCH /admin/api/v1/color-aliases", RunWithMUDLocked(
		doBasicAuth(apiV1PatchColorAlias),
	))
	mux.HandleFunc("DELETE /admin/api/v1/color-aliases/{alias}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteColorAlias),
	))

	// Color Patterns
	mux.HandleFunc("GET /admin/api/v1/colorpatterns", RunWithMUDLocked(
		doBasicAuth(apiV1GetColorPatterns),
	))
	mux.HandleFunc("POST /admin/api/v1/colorpatterns", RunWithMUDLocked(
		doBasicAuth(apiV1CreateColorPattern),
	))
	mux.HandleFunc("PATCH /admin/api/v1/colorpatterns", RunWithMUDLocked(
		doBasicAuth(apiV1PatchColorPatterns),
	))
	mux.HandleFunc("DELETE /admin/api/v1/colorpatterns", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteColorPattern),
	))

	// Mobs — script sub-route before wildcard {mobId}
	mux.HandleFunc("GET /admin/api/v1/mobs", RunWithMUDLocked(
		doBasicAuth(apiV1GetMobs),
	))
	mux.HandleFunc("POST /admin/api/v1/mobs", RunWithMUDLocked(
		doBasicAuth(apiV1CreateMob),
	))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1GetMobScript),
	))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1PutMobScript),
	))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/stock", RunWithMUDLocked(
		doBasicAuth(apiV1PutMobStock),
	))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}", RunWithMUDLocked(
		doBasicAuth(apiV1GetMob),
	))
	mux.HandleFunc("PATCH /admin/api/v1/mobs/{mobId}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchMob),
	))
	mux.HandleFunc("DELETE /admin/api/v1/mobs/{mobId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteMob),
	))

	// Zones
	mux.HandleFunc("GET /admin/api/v1/zones", RunWithMUDLocked(
		doBasicAuth(apiV1GetZones),
	))
	mux.HandleFunc("POST /admin/api/v1/zones", RunWithMUDLocked(
		doBasicAuth(apiV1CreateZone),
	))
	mux.HandleFunc("PATCH /admin/api/v1/zones/{zonename}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchZone),
	))
	mux.HandleFunc("DELETE /admin/api/v1/zones/{zonename}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteZone),
	))

	// Rooms — static sub-routes before wildcard {roomId}
	mux.HandleFunc("GET /admin/api/v1/rooms/biomes", RunWithMUDLocked(
		doBasicAuth(apiV1GetBiomes),
	))
	mux.HandleFunc("GET /admin/api/v1/rooms", RunWithMUDLocked(
		doBasicAuth(apiV1GetRooms),
	))
	mux.HandleFunc("POST /admin/api/v1/rooms", RunWithMUDLocked(
		doBasicAuth(apiV1CreateRoom),
	))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1GetRoomScript),
	))
	mux.HandleFunc("PUT /admin/api/v1/rooms/{roomId}/script", RunWithMUDLocked(
		doBasicAuth(apiV1PutRoomScript),
	))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}", RunWithMUDLocked(
		doBasicAuth(apiV1GetRoom),
	))
	mux.HandleFunc("PATCH /admin/api/v1/rooms/{roomId}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchRoom),
	))
	mux.HandleFunc("DELETE /admin/api/v1/rooms/{roomId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteRoom),
	))

	// Mutators
	mux.HandleFunc("GET /admin/api/v1/mutators", RunWithMUDLocked(
		doBasicAuth(apiV1GetMutators),
	))
	mux.HandleFunc("POST /admin/api/v1/mutators", RunWithMUDLocked(
		doBasicAuth(apiV1CreateMutator),
	))
	mux.HandleFunc("GET /admin/api/v1/mutators/{mutatorId}", RunWithMUDLocked(
		doBasicAuth(apiV1GetMutator),
	))
	mux.HandleFunc("PATCH /admin/api/v1/mutators/{mutatorId}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchMutator),
	))
	mux.HandleFunc("DELETE /admin/api/v1/mutators/{mutatorId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteMutator),
	))

	// Audio
	mux.HandleFunc("GET /admin/api/v1/audio", RunWithMUDLocked(
		doBasicAuth(apiV1GetAudio),
	))
	mux.HandleFunc("PATCH /admin/api/v1/audio", RunWithMUDLocked(
		doBasicAuth(apiV1PatchAudio),
	))

	// Keywords
	mux.HandleFunc("GET /admin/api/v1/keywords", RunWithMUDLocked(
		doBasicAuth(apiV1GetKeywords),
	))
	mux.HandleFunc("PATCH /admin/api/v1/keywords", RunWithMUDLocked(
		doBasicAuth(apiV1PatchKeywords),
	))

	// Races
	mux.HandleFunc("GET /admin/api/v1/races", RunWithMUDLocked(
		doBasicAuth(apiV1GetRaces),
	))
	mux.HandleFunc("POST /admin/api/v1/races", RunWithMUDLocked(
		doBasicAuth(apiV1CreateRace),
	))
	mux.HandleFunc("PATCH /admin/api/v1/races/{raceId}", RunWithMUDLocked(
		doBasicAuth(apiV1PatchRace),
	))
	mux.HandleFunc("DELETE /admin/api/v1/races/{raceId}", RunWithMUDLocked(
		doBasicAuth(apiV1DeleteRace),
	))

	// Spells — script sub-route before wildcard {spellId}
	mux.HandleFunc("GET /admin/api/v2/spells", RunWithMUDLocked(
		doBasicAuth(apiV2GetSpells),
	))
	mux.HandleFunc("POST /admin/api/v2/spells", RunWithMUDLocked(
		doBasicAuth(apiV2CreateSpell),
	))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}/script", RunWithMUDLocked(
		doBasicAuth(apiV2GetSpellScript),
	))
	mux.HandleFunc("PUT /admin/api/v2/spells/{spellId}/script", RunWithMUDLocked(
		doBasicAuth(apiV2PutSpellScript),
	))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}", RunWithMUDLocked(
		doBasicAuth(apiV2GetSpell),
	))
	mux.HandleFunc("PATCH /admin/api/v2/spells/{spellId}", RunWithMUDLocked(
		doBasicAuth(apiV2PatchSpell),
	))
	mux.HandleFunc("DELETE /admin/api/v2/spells/{spellId}", RunWithMUDLocked(
		doBasicAuth(apiV2DeleteSpell),
	))
}
