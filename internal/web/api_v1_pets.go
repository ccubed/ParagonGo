package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// GET /admin/api/v1/pets
func apiV1GetPets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[map[string]pets.Pet]{
		Success: true,
		Data:    pets.GetAllPetSpecs(),
	})
}

// POST /admin/api/v1/pets
func apiV1CreatePet(w http.ResponseWriter, r *http.Request) {
	var p pets.Pet
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}
	if strings.TrimSpace(p.Type) == "" {
		writeAPIError(w, http.StatusBadRequest, "type is required")
		return
	}
	if err := pets.CreatePetSpec(&p); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, APIResponse[pets.Pet]{
		Success: true,
		Data:    p,
	})
}

// PATCH /admin/api/v1/pets/{petname}
func apiV1PatchPet(w http.ResponseWriter, r *http.Request) {
	petName := strings.ToLower(strings.TrimSpace(r.PathValue("petname")))
	if petName == "" {
		writeAPIError(w, http.StatusBadRequest, "petname is required")
		return
	}

	all := pets.GetAllPetSpecs()
	existing, ok := all[petName]
	if !ok {
		writeAPIError(w, http.StatusNotFound, "pet type not found: "+petName)
		return
	}

	updated := existing
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}
	updated.Type = petName

	if err := pets.SavePetSpec(&updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[pets.Pet]{
		Success: true,
		Data:    updated,
	})
}

// GET /admin/api/v1/pets/ranks
func apiV1GetPetRanks(w http.ResponseWriter, r *http.Request) {
	byCombat, byUtility, byOverall := combat.RankPets()
	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"by_combat":  byCombat,
			"by_utility": byUtility,
			"by_overall": byOverall,
		},
	})
}

// DELETE /admin/api/v1/pets/{petname}
func apiV1DeletePet(w http.ResponseWriter, r *http.Request) {
	petName := strings.ToLower(strings.TrimSpace(r.PathValue("petname")))
	if petName == "" {
		writeAPIError(w, http.StatusBadRequest, "petname is required")
		return
	}

	if err := pets.DeletePetSpec(petName); err != nil {
		writeAPIError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/pets/{petname}/script
func apiV1GetPetScript(w http.ResponseWriter, r *http.Request) {
	petName := strings.ToLower(strings.TrimSpace(r.PathValue("petname")))
	if petName == "" {
		writeAPIError(w, http.StatusBadRequest, "petname is required")
		return
	}

	all := pets.GetAllPetSpecs()
	spec, ok := all[petName]
	if !ok {
		writeAPIError(w, http.StatusNotFound, "pet type not found: "+petName)
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": spec.GetScript(), "lang": scriptLangString(spec.GetScriptPath())},
	})
}

// PUT /admin/api/v1/pets/{petname}/script
func apiV1PutPetScript(w http.ResponseWriter, r *http.Request) {
	petName := strings.ToLower(strings.TrimSpace(r.PathValue("petname")))
	if petName == "" {
		writeAPIError(w, http.StatusBadRequest, "petname is required")
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

	if err := pets.SavePetScript(petName, body.Script, body.Lang); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Invalidate the cached VM so the next invocation picks up the new script
	scripting.InvalidatePetVM(petName)

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
