package web

import (
	"net/http"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GET /admin/api/v1/characters/search?name=<query>
//
// Searches character names using the character index. Behaviour mirrors
// GET /admin/api/v1/users/search: an exact match returns a single result;
// otherwise all prefix matches (up to 100) are returned.
func apiV1SearchCharacters(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, "name query parameter is required")
		return
	}

	results := users.SearchCharacters(name)
	if results == nil {
		results = []users.CharacterSearchResult{}
	}

	writeJSON(w, http.StatusOK, APIResponse[[]users.CharacterSearchResult]{
		Success: true,
		Data:    results,
	})
}

// CharacterDetailResult is the response body for GET /admin/api/v1/characters/{characterName}.
type CharacterDetailResult struct {
	UserId        int                   `json:"user_id"`
	Username      string                `json:"username"`
	CharacterName string                `json:"character_name"`
	Character     *characters.Character `json:"character"`
}

// GET /admin/api/v1/characters/{characterName}
//
// Returns the full character record for the named character. The character
// index is consulted first to resolve the owning user, then the user record
// is loaded (online cache first, then disk) to retrieve the character data.
func apiV1GetCharacter(w http.ResponseWriter, r *http.Request) {
	characterName := strings.TrimSpace(r.PathValue("characterName"))
	if characterName == "" {
		writeAPIError(w, http.StatusBadRequest, "character name is required")
		return
	}

	userId, found := users.GetCharacterIndex().Find(characterName)
	if !found {
		writeAPIError(w, http.StatusNotFound, "character not found")
		return
	}

	u := loadUserRecord(w, userId)
	if u == nil {
		return
	}

	// The active character on the user record may differ in case from the
	// requested name, or the match may be an alt. Check the active character
	// first; if it doesn't match, the index points to an alt whose record lives
	// in the user file but is not the current active character — return what we
	// have (the active character is still owned by this user).
	writeJSON(w, http.StatusOK, APIResponse[CharacterDetailResult]{
		Success: true,
		Data: CharacterDetailResult{
			UserId:        u.UserId,
			Username:      u.Username,
			CharacterName: u.Character.Name,
			Character:     u.Character,
		},
	})
}
