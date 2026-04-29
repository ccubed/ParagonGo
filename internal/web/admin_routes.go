package web

import (
	"net/http"
)

func registerAdminRoutes(mux *http.ServeMux) {
	// Static assets (non-HTML files) under /admin/ require authentication.
	mux.HandleFunc("GET /admin/{file}", doBasicAuth(serveAdminStaticFile))
	mux.HandleFunc("GET /admin/static/{path...}", doBasicAuth(serveAdminStaticFile))

	mux.HandleFunc("GET /admin/", RunWithMUDLocked(
		doBasicAuth(adminIndex),
	))
	mux.HandleFunc("GET /admin/https/", RunWithMUDLocked(
		doBasicAuth(httpsIndex),
	))

	mux.HandleFunc("GET /admin/config", RunWithMUDLocked(
		doBasicAuth(adminConfig),
	))
	mux.HandleFunc("GET /admin/config-api", RunWithMUDLocked(
		doBasicAuth(adminConfigAPI),
	))

	mux.HandleFunc("GET /admin/items", RunWithMUDLocked(
		doBasicAuth(adminItems),
	))
	mux.HandleFunc("GET /admin/items-rank", RunWithMUDLocked(
		doBasicAuth(adminItemsRank),
	))
	mux.HandleFunc("GET /admin/items-api", RunWithMUDLocked(
		doBasicAuth(adminItemsAPI),
	))
	mux.HandleFunc("GET /admin/items-attack-messages", RunWithMUDLocked(
		doBasicAuth(adminItemsAttackMessages),
	))
	mux.HandleFunc("GET /admin/items-attack-messages-api", RunWithMUDLocked(
		doBasicAuth(adminItemsAttackMessagesAPI),
	))

	mux.HandleFunc("GET /admin/buffs", RunWithMUDLocked(
		doBasicAuth(adminBuffs),
	))
	mux.HandleFunc("GET /admin/buffs-api", RunWithMUDLocked(
		doBasicAuth(adminBuffsAPI),
	))

	mux.HandleFunc("GET /admin/quests", RunWithMUDLocked(
		doBasicAuth(adminQuests),
	))
	mux.HandleFunc("GET /admin/quests-api", RunWithMUDLocked(
		doBasicAuth(adminQuestsAPI),
	))

	mux.HandleFunc("GET /admin/users", RunWithMUDLocked(
		doBasicAuth(adminUsers),
	))
	mux.HandleFunc("GET /admin/users-api", RunWithMUDLocked(
		doBasicAuth(adminUsersAPI),
	))

	mux.HandleFunc("GET /admin/color-tester", RunWithMUDLocked(
		doBasicAuth(adminColorTester),
	))

	mux.HandleFunc("GET /admin/color-aliases", RunWithMUDLocked(
		doBasicAuth(adminColorAliases),
	))
	mux.HandleFunc("GET /admin/color-aliases-api", RunWithMUDLocked(
		doBasicAuth(adminColorAliasesAPI),
	))

	mux.HandleFunc("GET /admin/colorpatterns", RunWithMUDLocked(
		doBasicAuth(adminColorPatterns),
	))
	mux.HandleFunc("GET /admin/colorpatterns-api", RunWithMUDLocked(
		doBasicAuth(adminColorPatternsAPI),
	))

	mux.HandleFunc("GET /admin/races", RunWithMUDLocked(
		doBasicAuth(adminRaces),
	))
	mux.HandleFunc("GET /admin/races-api", RunWithMUDLocked(
		doBasicAuth(adminRacesAPI),
	))

	mux.HandleFunc("GET /admin/keywords", RunWithMUDLocked(
		doBasicAuth(adminKeywords),
	))
	mux.HandleFunc("GET /admin/keywords-api", RunWithMUDLocked(
		doBasicAuth(adminKeywordsAPI),
	))

	mux.HandleFunc("GET /admin/mobs", RunWithMUDLocked(
		doBasicAuth(adminMobs),
	))
	mux.HandleFunc("GET /admin/mobs-api", RunWithMUDLocked(
		doBasicAuth(adminMobsAPI),
	))

	mux.HandleFunc("GET /admin/mutators", RunWithMUDLocked(
		doBasicAuth(adminMutators),
	))
	mux.HandleFunc("GET /admin/mutators-api", RunWithMUDLocked(
		doBasicAuth(adminMutatorsAPI),
	))

	mux.HandleFunc("GET /admin/rooms", RunWithMUDLocked(
		doBasicAuth(adminRooms),
	))
	mux.HandleFunc("GET /admin/rooms-api", RunWithMUDLocked(
		doBasicAuth(adminRoomsAPI),
	))

	mux.HandleFunc("GET /admin/stats", RunWithMUDLocked(
		doBasicAuth(adminStats),
	))
	mux.HandleFunc("GET /admin/stats-api", RunWithMUDLocked(
		doBasicAuth(adminStatsAPI),
	))

	mux.HandleFunc("GET /admin/spells", RunWithMUDLocked(
		doBasicAuth(adminSpells),
	))
	mux.HandleFunc("GET /admin/spells-api", RunWithMUDLocked(
		doBasicAuth(adminSpellsAPI),
	))

	mux.HandleFunc("GET /admin/audio", RunWithMUDLocked(
		doBasicAuth(adminAudio),
	))
	mux.HandleFunc("GET /admin/audio-api", RunWithMUDLocked(
		doBasicAuth(adminAudioAPI),
	))

	registerAdminAPIRoutes(mux)
}
