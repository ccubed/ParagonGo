package web

import (
	"encoding/json"
	"net/http"

	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

type spellListEntry struct {
	spells.SpellData
	HasScript bool `json:"HasScript"`
}

// GET /admin/api/v2/spells
func apiV2GetSpells(w http.ResponseWriter, r *http.Request) {
	allSpells := spells.GetAllSpells()
	result := make([]spellListEntry, 0, len(allSpells))
	for _, s := range allSpells {
		result = append(result, spellListEntry{
			SpellData: *s,
			HasScript: s.GetScript() != "",
		})
	}
	writeJSON(w, http.StatusOK, APIResponse[[]spellListEntry]{
		Success: true,
		Data:    result,
	})
}

// GET /admin/api/v2/spells/{spellId}
func apiV2GetSpell(w http.ResponseWriter, r *http.Request) {
	spellId := r.PathValue("spellId")
	spec := spells.GetSpell(spellId)
	if spec == nil {
		writeAPIError(w, http.StatusNotFound, "spell not found: "+spellId)
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*spells.SpellData]{
		Success: true,
		Data:    spec,
	})
}

// POST /admin/api/v2/spells
func apiV2CreateSpell(w http.ResponseWriter, r *http.Request) {
	var spec spells.SpellData
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if spec.SpellId == "" {
		writeAPIError(w, http.StatusBadRequest, "SpellId is required")
		return
	}

	if existing := spells.GetSpell(spec.SpellId); existing != nil {
		writeAPIError(w, http.StatusConflict, "spell already exists: "+spec.SpellId)
		return
	}

	if err := spells.SaveSpellSpec(&spec); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, APIResponse[*spells.SpellData]{
		Success: true,
		Data:    &spec,
	})
}

// PATCH /admin/api/v2/spells/{spellId}
func apiV2PatchSpell(w http.ResponseWriter, r *http.Request) {
	spellId := r.PathValue("spellId")
	existing := spells.GetSpell(spellId)
	if existing == nil {
		writeAPIError(w, http.StatusNotFound, "spell not found: "+spellId)
		return
	}

	updated := *existing
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	updated.SpellId = spellId

	if err := spells.SaveSpellSpec(&updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*spells.SpellData]{
		Success: true,
		Data:    &updated,
	})
}

// DELETE /admin/api/v2/spells/{spellId}
func apiV2DeleteSpell(w http.ResponseWriter, r *http.Request) {
	spellId := r.PathValue("spellId")
	if spells.GetSpell(spellId) == nil {
		writeAPIError(w, http.StatusNotFound, "spell not found: "+spellId)
		return
	}

	if err := spells.DeleteSpellSpec(spellId); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v2/spells/{spellId}/script
func apiV2GetSpellScript(w http.ResponseWriter, r *http.Request) {
	spellId := r.PathValue("spellId")
	spec := spells.GetSpell(spellId)
	if spec == nil {
		writeAPIError(w, http.StatusNotFound, "spell not found: "+spellId)
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": spec.GetScript(), "lang": scriptLangString(spec.GetScriptPath())},
	})
}

// PUT /admin/api/v2/spells/{spellId}/script
func apiV2PutSpellScript(w http.ResponseWriter, r *http.Request) {
	spellId := r.PathValue("spellId")
	if spells.GetSpell(spellId) == nil {
		writeAPIError(w, http.StatusNotFound, "spell not found: "+spellId)
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

	if err := spells.SaveSpellScript(spellId, body.Script, body.Lang); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scripting.InvalidateSpellVM(spellId)

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
