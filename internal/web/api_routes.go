package web

import (
	"net/http"
)

func registerAdminAPIRoutes(mux *http.ServeMux) {
	// Scripting
	mux.HandleFunc("GET /admin/api/v1/scripting/functions", doBasicAuth(RunWithMUDLocked(apiV1GetScriptFunctions)))

	// Telemetry
	mux.HandleFunc("GET /admin/api/v1/telemetry", doBasicAuth(RunWithMUDLocked(apiV1GetTelemetry)))
	mux.HandleFunc("DELETE /admin/api/v1/telemetry", doBasicAuth(RunWithMUDLocked(apiV1DeleteTelemetry)))
	mux.HandleFunc("POST /admin/api/v1/scripting/validate", doBasicAuth(RunWithMUDLocked(apiV1ValidateScript)))

	// Tags
	mux.HandleFunc("GET /admin/api/v1/tags", doBasicAuth(RunWithMUDLocked(apiV1GetTags)))

	// Stats
	mux.HandleFunc("GET /admin/api/v1/stats/memory", doBasicAuth(RunWithMUDLocked(apiV1GetStatsMemory)))

	// Connections
	mux.HandleFunc("GET /admin/api/v1/connections", doBasicAuth(RunWithMUDLocked(apiV1GetConnections)))

	// Config
	mux.HandleFunc("GET /admin/api/v1/config", doBasicAuth(RunWithMUDLocked(apiV1GetConfig)))
	mux.HandleFunc("PATCH /admin/api/v1/config", doBasicAuth(RunWithMUDLocked(RunInTestMode(apiV1PatchConfig))))

	// StatMods
	mux.HandleFunc("GET /admin/api/v1/statmods", doBasicAuth(RunWithMUDLocked(apiV1GetStatMods)))

	// Items - static sub-routes must be registered before the wildcard {itemId}
	// pattern so the Go 1.22 ServeMux prefers the more specific match.
	mux.HandleFunc("GET /admin/api/v1/items/types", doBasicAuth(RunWithMUDLocked(apiV1GetItemTypes)))
	mux.HandleFunc("GET /admin/api/v1/items/attack-messages", doBasicAuth(RunWithMUDLocked(apiV1GetItemAttackMessages)))
	mux.HandleFunc("GET /admin/api/v1/items/ranks/weapons", doBasicAuth(RunWithMUDLocked(apiV1GetItemRanksWeapons)))
	mux.HandleFunc("GET /admin/api/v1/items/ranks/armor", doBasicAuth(RunWithMUDLocked(apiV1GetItemRanksArmor)))
	mux.HandleFunc("PUT /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}", doBasicAuth(RunWithMUDLocked(apiV1PutItemAttackMessage)))
	mux.HandleFunc("DELETE /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}/{index}", doBasicAuth(RunWithMUDLocked(apiV1DeleteItemAttackMessage)))
	mux.HandleFunc("GET /admin/api/v1/items", doBasicAuth(RunWithMUDLocked(apiV1GetItems)))
	mux.HandleFunc("POST /admin/api/v1/items", doBasicAuth(RunWithMUDLocked(apiV1CreateItem)))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetItemScript)))
	mux.HandleFunc("PUT /admin/api/v1/items/{itemId}/script", doBasicAuth(RunWithMUDLocked(apiV1PutItemScript)))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}", doBasicAuth(RunWithMUDLocked(apiV1GetItem)))
	mux.HandleFunc("PATCH /admin/api/v1/items/{itemId}", doBasicAuth(RunWithMUDLocked(apiV1PatchItem)))
	mux.HandleFunc("DELETE /admin/api/v1/items/{itemId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteItem)))

	// Buffs
	mux.HandleFunc("GET /admin/api/v1/buffs", doBasicAuth(RunWithMUDLocked(apiV1GetBuffs)))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetBuffScript)))
	mux.HandleFunc("PUT /admin/api/v1/buffs/{buffId}/script", doBasicAuth(RunWithMUDLocked(apiV1PutBuffScript)))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}", doBasicAuth(RunWithMUDLocked(apiV1GetBuff)))
	mux.HandleFunc("PATCH /admin/api/v1/buffs/{buffId}", doBasicAuth(RunWithMUDLocked(apiV1PatchBuff)))
	mux.HandleFunc("DELETE /admin/api/v1/buffs/{buffId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteBuff)))

	// Quests
	mux.HandleFunc("GET /admin/api/v1/quests", doBasicAuth(RunWithMUDLocked(apiV1GetQuests)))
	mux.HandleFunc("PATCH /admin/api/v1/quests", doBasicAuth(RunWithMUDLocked(apiV1PatchQuest)))
	mux.HandleFunc("DELETE /admin/api/v1/quests/{questId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteQuest)))

	// Users - static sub-routes must be registered before any future wildcard
	// {userId} pattern.
	mux.HandleFunc("GET /admin/api/v1/users/search", doBasicAuth(RunWithMUDLocked(apiV1SearchUsers)))
	mux.HandleFunc("POST /admin/api/v1/users", doBasicAuth(RunWithMUDLocked(apiV1CreateUser)))
	mux.HandleFunc("GET /admin/api/v1/users/{userid}", doBasicAuth(RunWithMUDLocked(apiV1GetUser)))
	mux.HandleFunc("PATCH /admin/api/v1/users/{userid}", doBasicAuth(RunWithMUDLocked(apiV1PatchUser)))

	// Characters - static sub-routes must be registered before the wildcard
	// {characterName} pattern.
	mux.HandleFunc("GET /admin/api/v1/characters/search", doBasicAuth(RunWithMUDLocked(apiV1SearchCharacters)))
	mux.HandleFunc("GET /admin/api/v1/characters/{characterName}", doBasicAuth(RunWithMUDLocked(apiV1GetCharacter)))

	// Color Aliases
	mux.HandleFunc("GET /admin/api/v1/color-aliases", doBasicAuth(RunWithMUDLocked(apiV1GetColorAliases)))
	mux.HandleFunc("PATCH /admin/api/v1/color-aliases", doBasicAuth(RunWithMUDLocked(apiV1PatchColorAlias)))
	mux.HandleFunc("DELETE /admin/api/v1/color-aliases/{alias}", doBasicAuth(RunWithMUDLocked(apiV1DeleteColorAlias)))

	// Color Patterns
	mux.HandleFunc("GET /admin/api/v1/colorpatterns", doBasicAuth(RunWithMUDLocked(apiV1GetColorPatterns)))
	mux.HandleFunc("POST /admin/api/v1/colorpatterns", doBasicAuth(RunWithMUDLocked(apiV1CreateColorPattern)))
	mux.HandleFunc("PATCH /admin/api/v1/colorpatterns", doBasicAuth(RunWithMUDLocked(apiV1PatchColorPatterns)))
	mux.HandleFunc("DELETE /admin/api/v1/colorpatterns", doBasicAuth(RunWithMUDLocked(apiV1DeleteColorPattern)))

	// Mobs - script sub-route before wildcard {mobId}
	mux.HandleFunc("GET /admin/api/v1/mobs", doBasicAuth(RunWithMUDLocked(apiV1GetMobs)))
	mux.HandleFunc("POST /admin/api/v1/mobs", doBasicAuth(RunWithMUDLocked(apiV1CreateMob)))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetMobScript)))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/script", doBasicAuth(RunWithMUDLocked(apiV1PutMobScript)))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/stock", doBasicAuth(RunWithMUDLocked(apiV1PutMobStock)))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1GetMob)))
	mux.HandleFunc("PATCH /admin/api/v1/mobs/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1PatchMob)))
	mux.HandleFunc("DELETE /admin/api/v1/mobs/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteMob)))

	// Zones
	mux.HandleFunc("GET /admin/api/v1/zones", doBasicAuth(RunWithMUDLocked(apiV1GetZones)))
	mux.HandleFunc("POST /admin/api/v1/zones", doBasicAuth(RunWithMUDLocked(apiV1CreateZone)))
	mux.HandleFunc("PATCH /admin/api/v1/zones/{zonename}", doBasicAuth(RunWithMUDLocked(apiV1PatchZone)))
	mux.HandleFunc("DELETE /admin/api/v1/zones/{zonename}", doBasicAuth(RunWithMUDLocked(apiV1DeleteZone)))

	// Biomes
	mux.HandleFunc("GET /admin/api/v1/biomes", doBasicAuth(RunWithMUDLocked(apiV1GetBiomesV2)))
	mux.HandleFunc("POST /admin/api/v1/biomes", doBasicAuth(RunWithMUDLocked(apiV1CreateBiome)))
	mux.HandleFunc("PATCH /admin/api/v1/biomes/{biomeId}", doBasicAuth(RunWithMUDLocked(apiV1PatchBiome)))
	mux.HandleFunc("DELETE /admin/api/v1/biomes/{biomeId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteBiome)))

	// Mapper
	mux.HandleFunc("GET /admin/api/v1/mapper/rooms", doBasicAuth(RunWithMUDLocked(apiV1GetMapperAllRooms)))
	mux.HandleFunc("GET /admin/api/v1/mapper/zone/{zonename}", doBasicAuth(RunWithMUDLocked(apiV1GetMapperZone)))

	// Rooms - static sub-routes before wildcard {roomId}
	mux.HandleFunc("GET /admin/api/v1/rooms/biomes", doBasicAuth(RunWithMUDLocked(apiV1GetBiomes)))
	mux.HandleFunc("GET /admin/api/v1/rooms", doBasicAuth(RunWithMUDLocked(apiV1GetRooms)))
	mux.HandleFunc("POST /admin/api/v1/rooms", doBasicAuth(RunWithMUDLocked(apiV1CreateRoom)))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetRoomScript)))
	mux.HandleFunc("PUT /admin/api/v1/rooms/{roomId}/script", doBasicAuth(RunWithMUDLocked(apiV1PutRoomScript)))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}/instance", doBasicAuth(RunWithMUDLocked(apiV1GetRoomInstance)))
	mux.HandleFunc("PUT /admin/api/v1/rooms/{roomId}/instance", doBasicAuth(RunWithMUDLocked(apiV1PutRoomInstance)))
	mux.HandleFunc("DELETE /admin/api/v1/rooms/{roomId}/instance", doBasicAuth(RunWithMUDLocked(apiV1DeleteRoomInstance)))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}", doBasicAuth(RunWithMUDLocked(apiV1GetRoom)))
	mux.HandleFunc("PATCH /admin/api/v1/rooms/{roomId}", doBasicAuth(RunWithMUDLocked(apiV1PatchRoom)))
	mux.HandleFunc("DELETE /admin/api/v1/rooms/{roomId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteRoom)))

	// Mutators
	mux.HandleFunc("GET /admin/api/v1/mutators", doBasicAuth(RunWithMUDLocked(apiV1GetMutators)))
	mux.HandleFunc("POST /admin/api/v1/mutators", doBasicAuth(RunWithMUDLocked(apiV1CreateMutator)))
	mux.HandleFunc("GET /admin/api/v1/mutators/{mutatorId}", doBasicAuth(RunWithMUDLocked(apiV1GetMutator)))
	mux.HandleFunc("PATCH /admin/api/v1/mutators/{mutatorId}", doBasicAuth(RunWithMUDLocked(apiV1PatchMutator)))
	mux.HandleFunc("DELETE /admin/api/v1/mutators/{mutatorId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteMutator)))

	// Audio
	mux.HandleFunc("GET /admin/api/v1/audio", doBasicAuth(RunWithMUDLocked(apiV1GetAudio)))
	mux.HandleFunc("PATCH /admin/api/v1/audio", doBasicAuth(RunWithMUDLocked(apiV1PatchAudio)))

	// Keywords
	mux.HandleFunc("GET /admin/api/v1/keywords", doBasicAuth(RunWithMUDLocked(apiV1GetKeywords)))
	mux.HandleFunc("PATCH /admin/api/v1/keywords", doBasicAuth(RunWithMUDLocked(apiV1PatchKeywords)))

	// Races
	mux.HandleFunc("GET /admin/api/v1/races", doBasicAuth(RunWithMUDLocked(apiV1GetRaces)))
	mux.HandleFunc("POST /admin/api/v1/races", doBasicAuth(RunWithMUDLocked(apiV1CreateRace)))
	mux.HandleFunc("PATCH /admin/api/v1/races/{raceId}", doBasicAuth(RunWithMUDLocked(apiV1PatchRace)))
	mux.HandleFunc("DELETE /admin/api/v1/races/{raceId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteRace)))

	// Pets
	mux.HandleFunc("GET /admin/api/v1/pets/ranks", doBasicAuth(RunWithMUDLocked(apiV1GetPetRanks)))
	mux.HandleFunc("GET /admin/api/v1/pets", doBasicAuth(RunWithMUDLocked(apiV1GetPets)))
	mux.HandleFunc("POST /admin/api/v1/pets", doBasicAuth(RunWithMUDLocked(apiV1CreatePet)))
	mux.HandleFunc("PATCH /admin/api/v1/pets/{petname}", doBasicAuth(RunWithMUDLocked(apiV1PatchPet)))
	mux.HandleFunc("DELETE /admin/api/v1/pets/{petname}", doBasicAuth(RunWithMUDLocked(apiV1DeletePet)))

	// Conversations
	mux.HandleFunc("GET /admin/api/v1/conversations", doBasicAuth(RunWithMUDLocked(apiV1GetConversations)))
	mux.HandleFunc("GET /admin/api/v1/conversations/{zone}/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1GetConversation)))
	mux.HandleFunc("PUT /admin/api/v1/conversations/{zone}/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1PutConversation)))
	mux.HandleFunc("DELETE /admin/api/v1/conversations/{zone}/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1DeleteConversation)))

	// GameTime Calendars - static list route before wildcard {calendar}
	mux.HandleFunc("GET /admin/api/v1/gametime", doBasicAuth(RunWithMUDLocked(apiV1GetGameTime)))
	mux.HandleFunc("POST /admin/api/v1/gametime", doBasicAuth(RunWithMUDLocked(apiV1CreateGameTimeCalendar)))
	mux.HandleFunc("GET /admin/api/v1/gametime/{calendar}", doBasicAuth(RunWithMUDLocked(apiV1GetGameTimeCalendar)))
	mux.HandleFunc("PATCH /admin/api/v1/gametime/{calendar}", doBasicAuth(RunWithMUDLocked(apiV1PatchGameTimeCalendar)))
	mux.HandleFunc("DELETE /admin/api/v1/gametime/{calendar}", doBasicAuth(RunWithMUDLocked(apiV1DeleteGameTimeCalendar)))

	// Panels
	mux.HandleFunc("GET /admin/api/v1/panels", doBasicAuth(RunWithMUDLocked(apiV1GetPanels)))
	mux.HandleFunc("POST /admin/api/v1/panels/validate", doBasicAuth(RunWithMUDLocked(apiV1PostPanelValidate)))
	mux.HandleFunc("POST /admin/api/v1/panels/preview", doBasicAuth(RunWithMUDLocked(apiV1PostPanelPreview)))
	mux.HandleFunc("PUT /admin/api/v1/panels/{panelname...}", doBasicAuth(RunWithMUDLocked(apiV1PutPanel)))

	// Spells - script sub-route before wildcard {spellId}
	mux.HandleFunc("GET /admin/api/v2/spells", doBasicAuth(RunWithMUDLocked(apiV2GetSpells)))
	mux.HandleFunc("POST /admin/api/v2/spells", doBasicAuth(RunWithMUDLocked(apiV2CreateSpell)))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}/script", doBasicAuth(RunWithMUDLocked(apiV2GetSpellScript)))
	mux.HandleFunc("PUT /admin/api/v2/spells/{spellId}/script", doBasicAuth(RunWithMUDLocked(apiV2PutSpellScript)))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}", doBasicAuth(RunWithMUDLocked(apiV2GetSpell)))
	mux.HandleFunc("PATCH /admin/api/v2/spells/{spellId}", doBasicAuth(RunWithMUDLocked(apiV2PatchSpell)))
	mux.HandleFunc("DELETE /admin/api/v2/spells/{spellId}", doBasicAuth(RunWithMUDLocked(apiV2DeleteSpell)))
}
