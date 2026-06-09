package web

import (
	"net/http"
)

func registerAdminAPIRoutes(mux *http.ServeMux) {
	// All GET (read) API endpoints are accessible to any authenticated mod/admin.
	// Only mutating methods (POST, PATCH, PUT, DELETE) require a write permission.

	// Scripting
	mux.HandleFunc("GET /admin/api/v1/scripting/functions", doBasicAuth(RunWithMUDLocked(apiV1GetScriptFunctions)))
	mux.HandleFunc("GET /admin/api/v1/scripting/objecttypes", doBasicAuth(RunWithMUDLocked(apiV1GetScriptObjectTypes)))
	mux.HandleFunc("GET /admin/api/v1/scripting/types.d.ts", doBasicAuth(apiV1GetScriptingTypesDts))

	// Telemetry
	mux.HandleFunc("GET /admin/api/v1/telemetry", doBasicAuth(RunWithoutMUDLock(apiV1GetTelemetry)))
	mux.HandleFunc("DELETE /admin/api/v1/telemetry", doBasicAuth(RequirePermission("telemetry.write", RunWithoutMUDLock(apiV1DeleteTelemetry))))

	// Tags
	mux.HandleFunc("GET /admin/api/v1/tags", doBasicAuth(RunWithMUDLocked(apiV1GetTags)))

	// Stats
	mux.HandleFunc("GET /admin/api/v1/stats/memory", doBasicAuth(RunWithMUDLocked(apiV1GetStatsMemory)))

	// Connections
	mux.HandleFunc("GET /admin/api/v1/connections", doBasicAuth(RunWithMUDLocked(apiV1GetConnections)))

	// Config
	mux.HandleFunc("GET /admin/api/v1/config", doBasicAuth(RunWithMUDLocked(apiV1GetConfig)))
	mux.HandleFunc("PATCH /admin/api/v1/config", doBasicAuth(RequirePermission("config.write", RunWithMUDLocked(RunInTestMode(apiV1PatchConfig)))))

	// Progression preview
	mux.HandleFunc("GET /admin/api/v1/progression/preview", doBasicAuth(RunWithMUDLocked(apiV1GetProgressionPreview)))

	// StatMods
	mux.HandleFunc("GET /admin/api/v1/statmods", doBasicAuth(RunWithMUDLocked(apiV1GetStatMods)))

	// Items
	mux.HandleFunc("GET /admin/api/v1/items/equip-slots", doBasicAuth(RunWithMUDLocked(apiV1GetEquipSlots)))
	mux.HandleFunc("GET /admin/api/v1/items/types", doBasicAuth(RunWithMUDLocked(apiV1GetItemTypes)))
	mux.HandleFunc("GET /admin/api/v1/items/attack-messages", doBasicAuth(RunWithMUDLocked(apiV1GetItemAttackMessages)))
	mux.HandleFunc("GET /admin/api/v1/items/ranks/weapons", doBasicAuth(RunWithMUDLocked(apiV1GetItemRanksWeapons)))
	mux.HandleFunc("GET /admin/api/v1/items/ranks/armor", doBasicAuth(RunWithMUDLocked(apiV1GetItemRanksArmor)))
	mux.HandleFunc("PUT /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1PutItemAttackMessage))))
	mux.HandleFunc("PATCH /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}/{index}", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1PatchItemAttackMessage))))
	mux.HandleFunc("DELETE /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}/{index}", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1DeleteItemAttackMessage))))
	mux.HandleFunc("GET /admin/api/v1/items/attack-messages/{subtype}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetAttackMessagesYAML)))
	mux.HandleFunc("GET /admin/api/v1/items", doBasicAuth(RunWithMUDLocked(apiV1GetItems)))
	mux.HandleFunc("POST /admin/api/v1/items", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1CreateItem))))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetItemScript)))
	mux.HandleFunc("PUT /admin/api/v1/items/{itemId}/script", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1PutItemScript))))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetItemYAML)))
	mux.HandleFunc("GET /admin/api/v1/items/{itemId}", doBasicAuth(RunWithMUDLocked(apiV1GetItem)))
	mux.HandleFunc("PATCH /admin/api/v1/items/{itemId}", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1PatchItem))))
	mux.HandleFunc("DELETE /admin/api/v1/items/{itemId}", doBasicAuth(RequirePermission("items.write", RunWithMUDLocked(apiV1DeleteItem))))

	// Buffs
	mux.HandleFunc("GET /admin/api/v1/buffs", doBasicAuth(RunWithMUDLocked(apiV1GetBuffs)))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetBuffScript)))
	mux.HandleFunc("PUT /admin/api/v1/buffs/{buffId}/script", doBasicAuth(RequirePermission("buffs.write", RunWithMUDLocked(apiV1PutBuffScript))))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetBuffYAML)))
	mux.HandleFunc("GET /admin/api/v1/buffs/{buffId}", doBasicAuth(RunWithMUDLocked(apiV1GetBuff)))
	mux.HandleFunc("PATCH /admin/api/v1/buffs/{buffId}", doBasicAuth(RequirePermission("buffs.write", RunWithMUDLocked(apiV1PatchBuff))))
	mux.HandleFunc("DELETE /admin/api/v1/buffs/{buffId}", doBasicAuth(RequirePermission("buffs.write", RunWithMUDLocked(apiV1DeleteBuff))))

	// Quests
	mux.HandleFunc("GET /admin/api/v1/quests", doBasicAuth(RunWithMUDLocked(apiV1GetQuests)))
	mux.HandleFunc("GET /admin/api/v1/quests/{questId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetQuestYAML)))
	mux.HandleFunc("PATCH /admin/api/v1/quests", doBasicAuth(RequirePermission("quests.write", RunWithMUDLocked(apiV1PatchQuest))))
	mux.HandleFunc("DELETE /admin/api/v1/quests/{questId}", doBasicAuth(RequirePermission("quests.write", RunWithMUDLocked(apiV1DeleteQuest))))

	// Users
	mux.HandleFunc("GET /admin/api/v1/users/search", doBasicAuth(RunWithMUDLocked(apiV1SearchUsers)))
	mux.HandleFunc("POST /admin/api/v1/users", doBasicAuth(RequirePermission("users.write", RunWithMUDLocked(apiV1CreateUser))))
	mux.HandleFunc("GET /admin/api/v1/users/{userid}/permissions", doBasicAuth(RunWithMUDLocked(apiV1GetUserPermissions)))
	mux.HandleFunc("PUT /admin/api/v1/users/{userid}/permissions", doBasicAuth(RequirePermission("users.write", RunWithMUDLocked(apiV1PutUserPermissions))))
	mux.HandleFunc("GET /admin/api/v1/users/{userid}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetUserYAML)))
	mux.HandleFunc("GET /admin/api/v1/users/{userid}", doBasicAuth(RunWithMUDLocked(apiV1GetUser)))
	mux.HandleFunc("PATCH /admin/api/v1/users/{userid}", doBasicAuth(RequirePermission("users.write", RunWithMUDLocked(apiV1PatchUser))))

	// Permissions catalog
	mux.HandleFunc("GET /admin/api/v1/permissions", doBasicAuth(RunWithMUDLocked(apiV1GetPermissions)))

	// Characters
	mux.HandleFunc("GET /admin/api/v1/characters/search", doBasicAuth(RunWithMUDLocked(apiV1SearchCharacters)))
	mux.HandleFunc("GET /admin/api/v1/characters/{characterName}", doBasicAuth(RunWithMUDLocked(apiV1GetCharacter)))

	// Color Aliases
	mux.HandleFunc("GET /admin/api/v1/color-aliases/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetColorAliasesYAML)))
	mux.HandleFunc("GET /admin/api/v1/color-aliases", doBasicAuth(RunWithMUDLocked(apiV1GetColorAliases)))
	mux.HandleFunc("PATCH /admin/api/v1/color-aliases", doBasicAuth(RequirePermission("color-aliases.write", RunWithMUDLocked(apiV1PatchColorAlias))))
	mux.HandleFunc("DELETE /admin/api/v1/color-aliases/{alias}", doBasicAuth(RequirePermission("color-aliases.write", RunWithMUDLocked(apiV1DeleteColorAlias))))

	// Color Patterns
	mux.HandleFunc("GET /admin/api/v1/colorpatterns/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetColorPatternsYAML)))
	mux.HandleFunc("GET /admin/api/v1/colorpatterns", doBasicAuth(RunWithMUDLocked(apiV1GetColorPatterns)))
	mux.HandleFunc("POST /admin/api/v1/colorpatterns", doBasicAuth(RequirePermission("colorpatterns.write", RunWithMUDLocked(apiV1CreateColorPattern))))
	mux.HandleFunc("PATCH /admin/api/v1/colorpatterns", doBasicAuth(RequirePermission("colorpatterns.write", RunWithMUDLocked(apiV1PatchColorPatterns))))
	mux.HandleFunc("DELETE /admin/api/v1/colorpatterns", doBasicAuth(RequirePermission("colorpatterns.write", RunWithMUDLocked(apiV1DeleteColorPattern))))

	// Mobs
	mux.HandleFunc("GET /admin/api/v1/mobs/ranks", doBasicAuth(RunWithMUDLocked(apiV1GetMobRanks)))
	mux.HandleFunc("GET /admin/api/v1/mobs", doBasicAuth(RunWithMUDLocked(apiV1GetMobs)))
	mux.HandleFunc("POST /admin/api/v1/mobs", doBasicAuth(RequirePermission("mobs.write", RunWithMUDLocked(apiV1CreateMob))))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}/scripts", doBasicAuth(RunWithMUDLocked(apiV1GetMobScripts)))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}/scripts/{tag}", doBasicAuth(RunWithMUDLocked(apiV1GetMobScriptByTag)))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/scripts/{tag}", doBasicAuth(RequirePermission("mobs.write", RunWithMUDLocked(apiV1PutMobScriptByTag))))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetMobScript)))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/script", doBasicAuth(RequirePermission("mobs.write", RunWithMUDLocked(apiV1PutMobScript))))
	mux.HandleFunc("PUT /admin/api/v1/mobs/{mobId}/stock", doBasicAuth(RequirePermission("mobs.write", RunWithMUDLocked(apiV1PutMobStock))))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetMobYAML)))
	mux.HandleFunc("GET /admin/api/v1/mobs/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1GetMob)))
	mux.HandleFunc("PATCH /admin/api/v1/mobs/{mobId}", doBasicAuth(RequirePermission("mobs.write", RunWithMUDLocked(apiV1PatchMob))))
	mux.HandleFunc("DELETE /admin/api/v1/mobs/{mobId}", doBasicAuth(RequirePermission("mobs.write", RunWithMUDLocked(apiV1DeleteMob))))

	// Zones
	mux.HandleFunc("GET /admin/api/v1/zones", doBasicAuth(RunWithMUDLocked(apiV1GetZones)))
	mux.HandleFunc("POST /admin/api/v1/zones", doBasicAuth(RequirePermission("zones.write", RunWithMUDLocked(apiV1CreateZone))))
	mux.HandleFunc("POST /admin/api/v1/zones/{zonename}/rename", doBasicAuth(RequirePermission("zones.write", RunWithMUDLocked(apiV1RenameZone))))
	mux.HandleFunc("PATCH /admin/api/v1/zones/{zonename}", doBasicAuth(RequirePermission("zones.write", RunWithMUDLocked(apiV1PatchZone))))
	mux.HandleFunc("DELETE /admin/api/v1/zones/{zonename}", doBasicAuth(RequirePermission("zones.write", RunWithMUDLocked(apiV1DeleteZone))))

	// Biomes
	mux.HandleFunc("GET /admin/api/v1/biomes", doBasicAuth(RunWithMUDLocked(apiV1GetBiomesV2)))
	mux.HandleFunc("POST /admin/api/v1/biomes", doBasicAuth(RequirePermission("biomes.write", RunWithMUDLocked(apiV1CreateBiome))))
	mux.HandleFunc("GET /admin/api/v1/biomes/{biomeId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetBiomeYAML)))
	mux.HandleFunc("PATCH /admin/api/v1/biomes/{biomeId}", doBasicAuth(RequirePermission("biomes.write", RunWithMUDLocked(apiV1PatchBiome))))
	mux.HandleFunc("DELETE /admin/api/v1/biomes/{biomeId}", doBasicAuth(RequirePermission("biomes.write", RunWithMUDLocked(apiV1DeleteBiome))))

	// Mapper
	mux.HandleFunc("GET /admin/api/v1/mapper/rooms", doBasicAuth(RunWithMUDLocked(apiV1GetMapperAllRooms)))
	mux.HandleFunc("GET /admin/api/v1/mapper/zone/{zonename}", doBasicAuth(RunWithMUDLocked(apiV1GetMapperZone)))

	// Rooms
	mux.HandleFunc("GET /admin/api/v1/rooms/biomes", doBasicAuth(RunWithMUDLocked(apiV1GetBiomes)))
	mux.HandleFunc("GET /admin/api/v1/rooms", doBasicAuth(RunWithMUDLocked(apiV1GetRooms)))
	mux.HandleFunc("POST /admin/api/v1/rooms", doBasicAuth(RequirePermission("rooms.write", RunWithMUDLocked(apiV1CreateRoom))))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}/script", doBasicAuth(RunWithMUDLocked(apiV1GetRoomScript)))
	mux.HandleFunc("PUT /admin/api/v1/rooms/{roomId}/script", doBasicAuth(RequirePermission("rooms.write", RunWithMUDLocked(apiV1PutRoomScript))))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}/instance", doBasicAuth(RunWithMUDLocked(apiV1GetRoomInstance)))
	mux.HandleFunc("PUT /admin/api/v1/rooms/{roomId}/instance", doBasicAuth(RequirePermission("rooms.write", RunWithMUDLocked(apiV1PutRoomInstance))))
	mux.HandleFunc("DELETE /admin/api/v1/rooms/{roomId}/instance", doBasicAuth(RequirePermission("rooms.write", RunWithMUDLocked(apiV1DeleteRoomInstance))))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetRoomYAML)))
	mux.HandleFunc("GET /admin/api/v1/rooms/{roomId}", doBasicAuth(RunWithMUDLocked(apiV1GetRoom)))
	mux.HandleFunc("PATCH /admin/api/v1/rooms/{roomId}", doBasicAuth(RequirePermission("rooms.write", RunWithMUDLocked(apiV1PatchRoom))))
	mux.HandleFunc("DELETE /admin/api/v1/rooms/{roomId}", doBasicAuth(RequirePermission("rooms.write", RunWithMUDLocked(apiV1DeleteRoom))))

	// Mutators
	mux.HandleFunc("GET /admin/api/v1/mutators", doBasicAuth(RunWithMUDLocked(apiV1GetMutators)))
	mux.HandleFunc("POST /admin/api/v1/mutators", doBasicAuth(RequirePermission("mutators.write", RunWithMUDLocked(apiV1CreateMutator))))
	mux.HandleFunc("GET /admin/api/v1/mutators/{mutatorId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetMutatorYAML)))
	mux.HandleFunc("GET /admin/api/v1/mutators/{mutatorId}", doBasicAuth(RunWithMUDLocked(apiV1GetMutator)))
	mux.HandleFunc("PATCH /admin/api/v1/mutators/{mutatorId}", doBasicAuth(RequirePermission("mutators.write", RunWithMUDLocked(apiV1PatchMutator))))
	mux.HandleFunc("DELETE /admin/api/v1/mutators/{mutatorId}", doBasicAuth(RequirePermission("mutators.write", RunWithMUDLocked(apiV1DeleteMutator))))

	// Audio
	mux.HandleFunc("GET /admin/api/v1/audio/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetAudioYAML)))
	mux.HandleFunc("GET /admin/api/v1/audio", doBasicAuth(RunWithMUDLocked(apiV1GetAudio)))
	mux.HandleFunc("POST /admin/api/v1/audio", doBasicAuth(RequirePermission("audio.write", RunWithMUDLocked(apiV1CreateAudio))))
	mux.HandleFunc("PATCH /admin/api/v1/audio", doBasicAuth(RequirePermission("audio.write", RunWithMUDLocked(apiV1PatchAudio))))
	mux.HandleFunc("DELETE /admin/api/v1/audio/{identifier}", doBasicAuth(RequirePermission("audio.write", RunWithMUDLocked(apiV1DeleteAudio))))

	// Keywords
	mux.HandleFunc("GET /admin/api/v1/keywords/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetKeywordsYAML)))
	mux.HandleFunc("GET /admin/api/v1/keywords", doBasicAuth(RunWithMUDLocked(apiV1GetKeywords)))
	mux.HandleFunc("PATCH /admin/api/v1/keywords", doBasicAuth(RequirePermission("keywords.write", RunWithMUDLocked(apiV1PatchKeywords))))

	// Races
	mux.HandleFunc("GET /admin/api/v1/races", doBasicAuth(RunWithMUDLocked(apiV1GetRaces)))
	mux.HandleFunc("POST /admin/api/v1/races", doBasicAuth(RequirePermission("races.write", RunWithMUDLocked(apiV1CreateRace))))
	mux.HandleFunc("GET /admin/api/v1/races/{raceId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetRaceYAML)))
	mux.HandleFunc("PATCH /admin/api/v1/races/{raceId}", doBasicAuth(RequirePermission("races.write", RunWithMUDLocked(apiV1PatchRace))))
	mux.HandleFunc("DELETE /admin/api/v1/races/{raceId}", doBasicAuth(RequirePermission("races.write", RunWithMUDLocked(apiV1DeleteRace))))

	// Pets
	mux.HandleFunc("GET /admin/api/v1/pets/ranks", doBasicAuth(RunWithMUDLocked(apiV1GetPetRanks)))
	mux.HandleFunc("GET /admin/api/v1/pets", doBasicAuth(RunWithMUDLocked(apiV1GetPets)))
	mux.HandleFunc("POST /admin/api/v1/pets", doBasicAuth(RequirePermission("pets.write", RunWithMUDLocked(apiV1CreatePet))))
	mux.HandleFunc("GET /admin/api/v1/pets/{petname}/script", doBasicAuth(RunWithMUDLocked(apiV1GetPetScript)))
	mux.HandleFunc("PUT /admin/api/v1/pets/{petname}/script", doBasicAuth(RequirePermission("pets.write", RunWithMUDLocked(apiV1PutPetScript))))
	mux.HandleFunc("GET /admin/api/v1/pets/{petname}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetPetYAML)))
	mux.HandleFunc("PATCH /admin/api/v1/pets/{petname}", doBasicAuth(RequirePermission("pets.write", RunWithMUDLocked(apiV1PatchPet))))
	mux.HandleFunc("DELETE /admin/api/v1/pets/{petname}", doBasicAuth(RequirePermission("pets.write", RunWithMUDLocked(apiV1DeletePet))))

	// Conversations
	mux.HandleFunc("GET /admin/api/v1/conversations", doBasicAuth(RunWithMUDLocked(apiV1GetConversations)))
	mux.HandleFunc("GET /admin/api/v1/conversations/{zone}/{mobId}/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetConversationYAML)))
	mux.HandleFunc("GET /admin/api/v1/conversations/{zone}/{mobId}", doBasicAuth(RunWithMUDLocked(apiV1GetConversation)))
	mux.HandleFunc("PUT /admin/api/v1/conversations/{zone}/{mobId}", doBasicAuth(RequirePermission("conversations.write", RunWithMUDLocked(apiV1PutConversation))))
	mux.HandleFunc("DELETE /admin/api/v1/conversations/{zone}/{mobId}", doBasicAuth(RequirePermission("conversations.write", RunWithMUDLocked(apiV1DeleteConversation))))

	// GameTime
	mux.HandleFunc("GET /admin/api/v1/gametime", doBasicAuth(RunWithMUDLocked(apiV1GetGameTime)))
	mux.HandleFunc("POST /admin/api/v1/gametime", doBasicAuth(RequirePermission("gametime.write", RunWithMUDLocked(apiV1CreateGameTimeCalendar))))
	mux.HandleFunc("GET /admin/api/v1/gametime/yaml", doBasicAuth(RunWithMUDLocked(apiV1GetGameTimeYAML)))
	mux.HandleFunc("GET /admin/api/v1/gametime/{calendar}", doBasicAuth(RunWithMUDLocked(apiV1GetGameTimeCalendar)))
	mux.HandleFunc("PATCH /admin/api/v1/gametime/{calendar}", doBasicAuth(RequirePermission("gametime.write", RunWithMUDLocked(apiV1PatchGameTimeCalendar))))
	mux.HandleFunc("DELETE /admin/api/v1/gametime/{calendar}", doBasicAuth(RequirePermission("gametime.write", RunWithMUDLocked(apiV1DeleteGameTimeCalendar))))

	// Panels
	mux.HandleFunc("GET /admin/api/v1/panels", doBasicAuth(RunWithMUDLocked(apiV1GetPanels)))
	mux.HandleFunc("POST /admin/api/v1/panels/validate", doBasicAuth(RunWithMUDLocked(apiV1PostPanelValidate)))
	mux.HandleFunc("POST /admin/api/v1/panels/preview", doBasicAuth(RunWithMUDLocked(apiV1PostPanelPreview)))
	mux.HandleFunc("PUT /admin/api/v1/panels/{panelname...}", doBasicAuth(RequirePermission("panels.write", RunWithMUDLocked(apiV1PutPanel))))

	// Spells
	mux.HandleFunc("GET /admin/api/v2/spells", doBasicAuth(RunWithMUDLocked(apiV2GetSpells)))
	mux.HandleFunc("POST /admin/api/v2/spells", doBasicAuth(RequirePermission("spells.write", RunWithMUDLocked(apiV2CreateSpell))))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}/script", doBasicAuth(RunWithMUDLocked(apiV2GetSpellScript)))
	mux.HandleFunc("PUT /admin/api/v2/spells/{spellId}/script", doBasicAuth(RequirePermission("spells.write", RunWithMUDLocked(apiV2PutSpellScript))))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}/yaml", doBasicAuth(RunWithMUDLocked(apiV2GetSpellYAML)))
	mux.HandleFunc("GET /admin/api/v2/spells/{spellId}", doBasicAuth(RunWithMUDLocked(apiV2GetSpell)))
	mux.HandleFunc("PATCH /admin/api/v2/spells/{spellId}", doBasicAuth(RequirePermission("spells.write", RunWithMUDLocked(apiV2PatchSpell))))
	mux.HandleFunc("DELETE /admin/api/v2/spells/{spellId}", doBasicAuth(RequirePermission("spells.write", RunWithMUDLocked(apiV2DeleteSpell))))
}
