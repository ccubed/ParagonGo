package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// GET /admin/api/v1/buffs
// Returns all buff flags (flag -> description map) AND all buff specs.
func apiV1GetBuffs(w http.ResponseWriter, r *http.Request) {
	type buffsResponse struct {
		Flags map[buffs.Flag]string   `json:"flags"`
		Specs map[int]*buffs.BuffSpec `json:"specs"`
	}
	writeJSON(w, http.StatusOK, APIResponse[buffsResponse]{
		Success: true,
		Data: buffsResponse{
			Flags: buffs.GetAllFlags(),
			Specs: buffs.GetAllBuffSpecs(),
		},
	})
}

// resolveBuffId converts a path segment to an integer buff ID and writes an
// error response if it is not a valid integer or the buff does not exist.
func resolveBuffId(w http.ResponseWriter, idStr string) (int, bool) {
	buffId, err := strconv.Atoi(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "buffId must be an integer: "+idStr)
		return 0, false
	}
	if buffs.GetBuffSpec(buffId) == nil {
		writeAPIError(w, http.StatusNotFound, "buff not found: "+idStr)
		return 0, false
	}
	return buffId, true
}

// GET /admin/api/v1/buffs/{buffId}
func apiV1GetBuff(w http.ResponseWriter, r *http.Request) {
	buffId, ok := resolveBuffId(w, r.PathValue("buffId"))
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*buffs.BuffSpec]{
		Success: true,
		Data:    buffs.GetBuffSpec(buffId),
	})
}

// PATCH /admin/api/v1/buffs/{buffId}
func apiV1PatchBuff(w http.ResponseWriter, r *http.Request) {
	buffId, ok := resolveBuffId(w, r.PathValue("buffId"))
	if !ok {
		return
	}

	existing := buffs.GetBuffSpec(buffId)

	// Decode into a copy so only supplied fields are changed.
	updated := *existing
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	// Preserve the canonical ID.
	updated.BuffId = buffId

	if err := buffs.SaveBuffSpec(&updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*buffs.BuffSpec]{
		Success: true,
		Data:    &updated,
	})
}

// DELETE /admin/api/v1/buffs/{buffId}
func apiV1DeleteBuff(w http.ResponseWriter, r *http.Request) {
	buffId, ok := resolveBuffId(w, r.PathValue("buffId"))
	if !ok {
		return
	}

	if err := buffs.DeleteBuffSpec(buffId); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/buffs/{buffId}/script
func apiV1GetBuffScript(w http.ResponseWriter, r *http.Request) {
	buffId, ok := resolveBuffId(w, r.PathValue("buffId"))
	if !ok {
		return
	}

	spec := buffs.GetBuffSpec(buffId)
	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": spec.GetScript(), "lang": scriptLangString(spec.GetScriptPath())},
	})
}

// PUT /admin/api/v1/buffs/{buffId}/script
func apiV1PutBuffScript(w http.ResponseWriter, r *http.Request) {
	buffId, ok := resolveBuffId(w, r.PathValue("buffId"))
	if !ok {
		return
	}

	var body struct {
		Script string `json:"script"`
		Lang   string `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := buffs.SaveBuffScript(buffId, body.Script, body.Lang); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scripting.InvalidateBuffVM(buffId)

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
