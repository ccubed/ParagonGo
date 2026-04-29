package web

import (
	"net/http"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// serveAdminTemplate is a helper that parses and executes a named admin HTML
// template, merging any extra data into the standard template data map.
func serveAdminTemplate(w http.ResponseWriter, r *http.Request, filename string, extra map[string]any) {
	adminHtml := configs.GetFilePathsConfig().AdminHtml.String()

	tmpl, err := template.New(filename).Funcs(funcMap).ParseFiles(
		adminHtml+"/_header.html",
		adminHtml+"/"+filename,
		adminHtml+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML ERROR", "error", err)
		http.Error(w, "Error parsing template files", http.StatusInternalServerError)
		return
	}

	templateData := map[string]any{
		"CONFIG": configs.GetConfig(),
		"STATS":  GetStats(),
		"NAV":    buildAdminNav(),
	}
	for k, v := range extra {
		templateData[k] = v
	}

	w.Header().Set("Cache-Control", "no-store")
	if err := tmpl.Execute(w, templateData); err != nil {
		mudlog.Error("HTML ERROR", "action", "Execute", "error", err)
	}
}

func adminItemsRank(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "items-rank.html", nil)
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

func adminRooms(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "rooms.html", nil)
}

func adminRoomsAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "rooms-api.html", nil)
}
