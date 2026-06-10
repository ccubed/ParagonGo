package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/scripting"
)

type mobListEntry struct {
	mobs.Mob
	HasScript bool `json:"HasScript"`
}

// GET /admin/api/v1/mobs
func apiV1GetMobs(w http.ResponseWriter, r *http.Request) {
	specs := mobs.GetAllMobSpecs()
	result := make([]mobListEntry, len(specs))
	for i, s := range specs {
		result[i] = mobListEntry{
			Mob:       s,
			HasScript: s.HasAnyScript(),
		}
	}
	writeJSON(w, http.StatusOK, APIResponse[[]mobListEntry]{
		Success: true,
		Data:    result,
	})
}

// GET /admin/api/v1/mobs/ranks
func apiV1GetMobRanks(w http.ResponseWriter, r *http.Request) {
	byThreat, byLoot, byDefense := combat.RankMobs()
	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"by_threat":  byThreat,
			"by_loot":    byLoot,
			"by_defense": byDefense,
		},
	})
}

func resolveMobId(w http.ResponseWriter, idOrName string) mobs.MobId {
	if id, err := strconv.Atoi(idOrName); err == nil {
		if spec := mobs.GetMobSpec(mobs.MobId(id)); spec != nil {
			return mobs.MobId(id)
		}
	}
	mobId := mobs.MobIdByName(idOrName)
	if mobId == 0 {
		writeAPIError(w, http.StatusNotFound, "mob not found: "+idOrName)
	}
	return mobId
}

// GET /admin/api/v1/mobs/{mobId}
func apiV1GetMob(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	spec := mobs.GetMobSpec(mobId)
	writeJSON(w, http.StatusOK, APIResponse[*mobs.Mob]{
		Success: true,
		Data:    spec,
	})
}

// POST /admin/api/v1/mobs
func apiV1CreateMob(w http.ResponseWriter, r *http.Request) {
	var spec mobs.Mob
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if spec.Character.Name == "" {
		writeAPIError(w, http.StatusBadRequest, "Character.Name is required")
		return
	}

	newId, err := mobs.CreateNewMobFile(spec, "")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	created := mobs.GetMobSpec(newId)
	writeJSON(w, http.StatusCreated, APIResponse[*mobs.Mob]{
		Success: true,
		Data:    created,
	})
}

// PATCH /admin/api/v1/mobs/{mobId}
func apiV1PatchMob(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	existing := mobs.GetMobSpec(mobId)
	if existing == nil {
		writeAPIError(w, http.StatusNotFound, "mob not found: "+strconv.Itoa(int(mobId)))
		return
	}

	updated := *existing

	// Clear the map-typed character collections the editor fully manages before
	// decoding. json.Unmarshal MERGES into existing maps rather than replacing
	// them, so a removed spell would otherwise persist. Resetting on the copy
	// gives the payload full replace semantics. (Character is a value on the
	// Mob, so this does not disturb the live spec's map.) Only SpellBook is
	// reset because that is the only map the mob editor submits — clearing
	// Skills here would drop skills the editor never sends.
	updated.Character.SpellBook = nil

	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	updated.MobId = mobId

	if err := mobs.SaveMobSpec(&updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*mobs.Mob]{
		Success: true,
		Data:    &updated,
	})
}

// DELETE /admin/api/v1/mobs/{mobId}
func apiV1DeleteMob(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	if err := mobs.DeleteMobSpec(mobId); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/mobs/{mobId}/script
func apiV1GetMobScript(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	spec := mobs.GetMobSpec(mobId)
	if spec == nil {
		writeAPIError(w, http.StatusNotFound, "mob not found")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": spec.GetScript(), "lang": scriptLangString(spec.GetScriptPath())},
	})
}

// PUT /admin/api/v1/mobs/{mobId}/script
func apiV1PutMobScript(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
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

	if err := mobs.SaveMobScript(mobId, body.Script, body.Lang); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scripting.InvalidateMobVMById(int(mobId))

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/mobs/{mobId}/scripts
func apiV1GetMobScripts(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	spec := mobs.GetMobSpec(mobId)
	if spec == nil {
		writeAPIError(w, http.StatusNotFound, "mob not found")
		return
	}

	tags := spec.GetAllScriptTags()
	if tags == nil {
		tags = []string{}
	}

	writeJSON(w, http.StatusOK, APIResponse[[]string]{
		Success: true,
		Data:    tags,
	})
}

// GET /admin/api/v1/mobs/{mobId}/scripts/{tag}
func apiV1GetMobScriptByTag(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	tag := r.PathValue("tag")

	script, err := mobs.GetMobScriptForTag(mobId, tag)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lang := "js"
	if spec := mobs.GetMobSpec(mobId); spec != nil {
		lang = scriptLangString(spec.GetScriptPathForTag(tag))
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": script, "tag": tag, "lang": lang},
	})
}

// PUT /admin/api/v1/mobs/{mobId}/scripts/{tag}
func apiV1PutMobScriptByTag(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	tag := r.PathValue("tag")

	var body struct {
		Script string `json:"script"`
		Lang   string `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := mobs.SaveMobScriptForTag(mobId, tag, body.Script, body.Lang); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	scripting.InvalidateMobVMById(int(mobId))

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// PUT /admin/api/v1/mobs/{mobId}/stock
func apiV1PutMobStock(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("mobId")
	mobId := resolveMobId(w, idOrName)
	if mobId == 0 {
		return
	}

	var entry characters.ShopItem
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if entry.Price < 0 {
		writeAPIError(w, http.StatusBadRequest, "price must be non-negative")
		return
	}
	if entry.Quantity < -1 {
		writeAPIError(w, http.StatusBadRequest, "quantity must be -1 or greater")
		return
	}
	if entry.ItemId != 0 && items.GetItemSpec(entry.ItemId) == nil {
		writeAPIError(w, http.StatusBadRequest, "item_id does not exist: "+strconv.Itoa(entry.ItemId))
		return
	}
	if entry.MobId != 0 && mobs.GetMobSpec(mobs.MobId(entry.MobId)) == nil {
		writeAPIError(w, http.StatusBadRequest, "mob_id does not exist: "+strconv.Itoa(entry.MobId))
		return
	}
	if entry.BuffId != 0 && buffs.GetBuffSpec(entry.BuffId) == nil {
		writeAPIError(w, http.StatusBadRequest, "buff_id does not exist: "+strconv.Itoa(entry.BuffId))
		return
	}

	if err := mobs.StockMobShop(mobId, entry); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
