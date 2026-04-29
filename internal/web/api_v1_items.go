package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// GET /admin/api/v1/items/ranks
// Returns weapon rankings computed by the combat engine against a neutral
// equal-stat opponent. Three sorted views are returned: by raw DPS, by
// adjusted DPS (two-handed and wait-round penalties applied), and by max
// single-hit damage.
func apiV1GetItemRanks(w http.ResponseWriter, r *http.Request) {
	byDPS, byAdjDPS, byMaxDmg := combat.RankWeapons()
	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"by_dps":     byDPS,
			"by_adj_dps": byAdjDPS,
			"by_max_dmg": byMaxDmg,
		},
	})
}

// GET /admin/api/v1/items/types
func apiV1GetItemTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[map[string]any]{
		Success: true,
		Data: map[string]any{
			"types":    items.ItemTypes(),
			"subtypes": items.ItemSubtypes(),
		},
	})
}

// GET /admin/api/v1/items/attack-messages
func apiV1GetItemAttackMessages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[map[items.ItemSubType]*items.WeaponAttackMessageGroup]{
		Success: true,
		Data:    items.GetAllAttackMessages(),
	})
}

type itemListEntry struct {
	items.ItemSpec
	HasScript bool `json:"HasScript"`
}

// GET /admin/api/v1/items
func apiV1GetItems(w http.ResponseWriter, r *http.Request) {
	specs := items.GetAllItemSpecs()
	result := make([]itemListEntry, len(specs))
	for i, s := range specs {
		result[i] = itemListEntry{
			ItemSpec:  s,
			HasScript: s.GetScript() != "",
		}
	}
	writeJSON(w, http.StatusOK, APIResponse[[]itemListEntry]{
		Success: true,
		Data:    result,
	})
}

// POST /admin/api/v1/items
func apiV1CreateItem(w http.ResponseWriter, r *http.Request) {
	var spec items.ItemSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	newId, err := items.CreateNewItemFile(spec)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	created := items.GetItemSpec(newId)
	writeJSON(w, http.StatusCreated, APIResponse[*items.ItemSpec]{
		Success: true,
		Data:    created,
	})
}

// resolveItemId resolves a path segment that is either a numeric item ID or an
// item name.  Returns 0 and writes an error response if not found.
func resolveItemId(w http.ResponseWriter, idOrName string) int {
	itemId := items.FindItem(idOrName)
	if itemId == 0 {
		writeAPIError(w, http.StatusNotFound, "item not found: "+idOrName)
	}
	return itemId
}

// GET /admin/api/v1/items/{itemId}
func apiV1GetItem(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("itemId")
	itemId := resolveItemId(w, idOrName)
	if itemId == 0 {
		return
	}

	spec := items.GetItemSpec(itemId)
	writeJSON(w, http.StatusOK, APIResponse[*items.ItemSpec]{
		Success: true,
		Data:    spec,
	})
}

// PATCH /admin/api/v1/items/{itemId}
func apiV1PatchItem(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("itemId")
	itemId := resolveItemId(w, idOrName)
	if itemId == 0 {
		return
	}

	existing := items.GetItemSpec(itemId)
	if existing == nil {
		writeAPIError(w, http.StatusNotFound, "item not found: "+strconv.Itoa(itemId))
		return
	}

	// Decode into a copy so we only apply supplied fields.
	updated := *existing
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	// Preserve the canonical ID; callers cannot change it via PATCH.
	updated.ItemId = itemId

	if err := items.SaveItemSpec(&updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[*items.ItemSpec]{
		Success: true,
		Data:    &updated,
	})
}

// DELETE /admin/api/v1/items/{itemId}
func apiV1DeleteItem(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("itemId")
	itemId := resolveItemId(w, idOrName)
	if itemId == 0 {
		return
	}

	if err := items.DeleteItemSpec(itemId); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// GET /admin/api/v1/items/{itemId}/script
func apiV1GetItemScript(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("itemId")
	itemId := resolveItemId(w, idOrName)
	if itemId == 0 {
		return
	}

	spec := items.GetItemSpec(itemId)
	if spec == nil {
		writeAPIError(w, http.StatusNotFound, "item not found")
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[map[string]string]{
		Success: true,
		Data:    map[string]string{"script": spec.GetScript()},
	})
}

// PUT /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}
func apiV1PutItemAttackMessage(w http.ResponseWriter, r *http.Request) {
	subtype := items.ItemSubType(r.PathValue("subtype"))
	intensity := items.Intensity(r.PathValue("intensity"))
	proximity := r.PathValue("proximity")
	target := r.PathValue("target")

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}
	if body.Message == "" {
		writeAPIError(w, http.StatusBadRequest, "message is required")
		return
	}

	if err := items.AddAttackMessage(subtype, intensity, proximity, target, body.Message); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// DELETE /admin/api/v1/items/attack-messages/{subtype}/{intensity}/{proximity}/{target}/{index}
func apiV1DeleteItemAttackMessage(w http.ResponseWriter, r *http.Request) {
	subtype := items.ItemSubType(r.PathValue("subtype"))
	intensity := items.Intensity(r.PathValue("intensity"))
	proximity := r.PathValue("proximity")
	target := r.PathValue("target")

	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid index: "+r.PathValue("index"))
		return
	}

	if err := items.DeleteAttackMessage(subtype, intensity, proximity, target, index); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}

// PUT /admin/api/v1/items/{itemId}/script
func apiV1PutItemScript(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("itemId")
	itemId := resolveItemId(w, idOrName)
	if itemId == 0 {
		return
	}

	var body struct {
		Script string `json:"script"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := items.SaveItemScript(itemId, body.Script); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
