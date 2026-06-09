package web

import (
	"net/http"

	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// scriptLangString maps a resolved script file path to the editor language
// identifier ("js" or "lua"). An empty path or unknown extension defaults to
// "js" so existing JavaScript scripts behave unchanged.
func scriptLangString(path string) string {
	if scripting.LangFromPath(path) == scripting.LangLua {
		return "lua"
	}
	return "js"
}

// GET /admin/api/v1/scripting/functions
func apiV1GetScriptFunctions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[*scripting.ScriptFunctionsSchema]{
		Success: true,
		Data:    scripting.GetScriptFunctionsSchema(),
	})
}

// GET /admin/api/v1/scripting/objecttypes
// Returns the structured engine object-type model (method lists for
// ActorObject, RoomObject, ItemObject, PetObject, PartyObject, ContainerObject). The Lua
// editor uses this together with the function schema to provide type-aware,
// context-sensitive completions and hovers, mirroring the JavaScript .d.ts.
func apiV1GetScriptObjectTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[*scripting.ScriptObjectTypes]{
		Success: true,
		Data:    scripting.GetScriptObjectTypes(),
	})
}
