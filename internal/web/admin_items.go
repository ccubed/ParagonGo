package web

import (
	"net/http"
	"strings"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// pageWritePermissions maps admin page URL paths to the write permission key
// that governs mutations on that page. Pages without a write permission (e.g.
// read-only views) are not listed.
var pageWritePermissions = map[string]string{
	"/admin/audio":                 "audio.write",
	"/admin/biomes":                "biomes.write",
	"/admin/buffs":                 "buffs.write",
	"/admin/color-aliases":         "color-aliases.write",
	"/admin/colorpatterns":         "colorpatterns.write",
	"/admin/config":                "config.write",
	"/admin/config-wizard":         "config.write",
	"/admin/progression":           "config.write",
	"/admin/conversations":         "conversations.write",
	"/admin/gametime":              "gametime.write",
	"/admin/items":                 "items.write",
	"/admin/items-attack-messages": "items.write",
	"/admin/keywords":              "keywords.write",
	"/admin/mobs":                  "mobs.write",
	"/admin/mutators":              "mutators.write",
	"/admin/panels":                "panels.write",
	"/admin/pets":                  "pets.write",
	"/admin/quests":                "quests.write",
	"/admin/races":                 "races.write",
	"/admin/rooms":                 "rooms.write",
	"/admin/spells":                "spells.write",
	"/admin/telemetry":             "telemetry.write",
	"/admin/users":                 "users.write",
	"/admin/mapper":                "rooms.write",
}

// pageReadOnly returns true when the authenticated user is a mod who lacks the
// write permission for the current admin page. Admins always have full access.
func pageReadOnly(r *http.Request) bool {
	u := GetAuthedUser(r)
	if u == nil || u.Role == users.RoleAdmin {
		return false
	}
	writeKey, ok := pageWritePermissions[strings.TrimRight(r.URL.Path, "/")]
	if !ok {
		return false
	}
	return !u.HasPermission(writeKey)
}

// serveAdminTemplate is a helper that parses and executes a named admin HTML
// template, merging any extra data into the standard template data map.
func serveAdminTemplate(w http.ResponseWriter, r *http.Request, filename string, extra map[string]any) {
	adminHtml := configs.GetFilePathsConfig().AdminHtml.String()

	tmpl, err := template.New(filename).Funcs(funcMap).ParseFiles(
		adminHtml+"/_header.html",
		adminHtml+"/_nav.html",
		adminHtml+"/"+filename,
		adminHtml+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML ERROR", "error", err)
		http.Error(w, "Error parsing template files", http.StatusInternalServerError)
		return
	}

	writeKey := pageWritePermissions[strings.TrimRight(r.URL.Path, "/")]
	templateData := map[string]any{
		"CONFIG":           configs.GetConfig(),
		"STATS":            GetStats(),
		"NAV":              buildAdminNav(),
		"AUTHED_USER":      GetAuthedUser(r),
		"WRITE_PERMISSION": writeKey,
		"READ_ONLY":        pageReadOnly(r),
	}
	for k, v := range extra {
		templateData[k] = v
	}

	w.Header().Set("Cache-Control", "no-store")
	if err := tmpl.Execute(w, templateData); err != nil {
		mudlog.Error("HTML ERROR", "action", "Execute", "error", err)
	}
}

func adminItemsRankWeapons(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items-rank-weapons.html", nil)
}

func adminItems(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items.html", nil)
}

func adminItemsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items-api.html", nil)
}

func adminItemsAttackMessages(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items-attack-messages.html", nil)
}

func adminItemsAttackMessagesAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items-attack-messages-api.html", nil)
}

func adminItemsRankArmor(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items-rank-armor.html", nil)
}

func adminBuffs(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "buffs.html", nil)
}

func adminBuffsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "buffs-api.html", nil)
}

func adminQuests(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "quests.html", nil)
}

func adminQuestsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "quests-api.html", nil)
}

func adminUsers(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "users.html", nil)
}

func adminUsersAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "users-api.html", nil)
}

func adminColorPatterns(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "colorpatterns.html", nil)
}

func adminColorPatternsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "colorpatterns-api.html", nil)
}

func adminColorAliases(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "color-aliases.html", nil)
}

func adminColorAliasesAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "color-aliases-api.html", nil)
}

func adminColorTester(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "color-tester.html", nil)
}

func adminRaces(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "races.html", nil)
}

func adminRacesAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "races-api.html", nil)
}

func adminKeywords(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "keywords.html", nil)
}

func adminKeywordsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "keywords-api.html", nil)
}

func adminMobs(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "mobs.html", nil)
}

func adminMobsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "mobs-api.html", nil)
}

func adminPets(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "pets.html", nil)
}

func adminPetsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "pets-api.html", nil)
}

func adminPetsRanks(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "pets-ranks.html", nil)
}

func adminMobRankings(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "mob-rankings.html", nil)
}

func adminSpells(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "spells.html", nil)
}

func adminSpellsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "spells-api.html", nil)
}

func adminMutators(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "mutators.html", nil)
}

func adminMutatorsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "mutators-api.html", nil)
}

func adminMapper(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "mapper.html", nil)
}

func adminRooms(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "rooms.html", nil)
}

func adminRoomsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "rooms-api.html", nil)
}

func adminBiomes(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "biomes.html", nil)
}

func adminBiomesAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "biomes-api.html", nil)
}

func adminConversations(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "conversations.html", nil)
}

func adminConversationsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "conversations-api.html", nil)
}

func adminGameTime(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "gametime.html", nil)
}

func adminGameTimeAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "gametime-api.html", nil)
}
