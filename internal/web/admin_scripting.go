package web

import "net/http"

func adminScriptingAPI(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-api.html", nil)
}

func adminScripting(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting.html", nil)
}

func adminScriptingRooms(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-rooms.html", nil)
}

func adminScriptingMobs(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-mobs.html", nil)
}

func adminScriptingItems(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-items.html", nil)
}

func adminScriptingBuffs(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-buffs.html", nil)
}

func adminScriptingSpells(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-spells.html", nil)
}

func adminScriptingPets(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-pets.html", nil)
}

func adminScriptingFunctions(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-functions.html", nil)
}

func adminScriptingObjects(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "scripting-objects.html", nil)
}

func adminDocsCoding(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "docs-coding.html", nil)
}

func adminDocsModules(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "docs-modules.html", nil)
}

func adminDocsBackups(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "docs-backups.html", nil)
}

func adminDocsAWS(w http.ResponseWriter, r *http.Request) {
	serveAdminTemplate(w, r, "docs-aws.html", nil)
}
