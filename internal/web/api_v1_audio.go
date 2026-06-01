package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/audio"
)

type audioAPIResponse struct {
	Sounds     map[string]audio.AudioConfig `json:"sounds"`
	Music      []string                     `json:"music"`
	SoundFiles []string                     `json:"soundFiles"`
}

// GET /admin/api/v1/audio
func apiV1GetAudio(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse[audioAPIResponse]{
		Success: true,
		Data: audioAPIResponse{
			Sounds:     audio.GetAllAudio(),
			Music:      audio.GetMusicFiles(),
			SoundFiles: audio.GetSoundFiles(),
		},
	})
}

// POST /admin/api/v1/audio
// Body: {"identifier": "my-sound", "description": "...", "filepath": "...", "volume": 80, "tags": ["combat"]}
// Creates a new sound entry. Returns 409 if the identifier already exists.
func apiV1CreateAudio(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Identifier  string   `json:"identifier"`
		Description string   `json:"description"`
		FilePath    string   `json:"filepath"`
		Volume      int      `json:"volume"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	body.Identifier = strings.TrimSpace(body.Identifier)
	if body.Identifier == "" {
		writeAPIError(w, http.StatusBadRequest, "identifier is required")
		return
	}

	current := audio.GetAllAudio()
	if _, exists := current[body.Identifier]; exists {
		writeAPIError(w, http.StatusConflict, "identifier already exists: "+body.Identifier)
		return
	}

	current[body.Identifier] = audio.AudioConfig{
		Description: body.Description,
		FilePath:    body.FilePath,
		Volume:      body.Volume,
		Tags:        body.Tags,
	}

	if err := audio.SaveAudio(current); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[audioAPIResponse]{
		Success: true,
		Data: audioAPIResponse{
			Sounds:     audio.GetAllAudio(),
			Music:      audio.GetMusicFiles(),
			SoundFiles: audio.GetSoundFiles(),
		},
	})
}

// PATCH /admin/api/v1/audio
// Body: {"identifier": {"description":"...","filepath":"...","volume":80}, ...}
// Replaces the entire sounds configuration.
func apiV1PatchAudio(w http.ResponseWriter, r *http.Request) {
	var patches map[string]audio.AudioConfig
	if err := json.NewDecoder(r.Body).Decode(&patches); err != nil {
		writeAPIError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return
	}

	if err := audio.SaveAudio(patches); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[audioAPIResponse]{
		Success: true,
		Data: audioAPIResponse{
			Sounds:     audio.GetAllAudio(),
			Music:      audio.GetMusicFiles(),
			SoundFiles: audio.GetSoundFiles(),
		},
	})
}

// DELETE /admin/api/v1/audio/{identifier}
// Removes a single sound entry from audio.yaml.
func apiV1DeleteAudio(w http.ResponseWriter, r *http.Request) {
	identifier := r.PathValue("identifier")
	if identifier == "" {
		writeAPIError(w, http.StatusBadRequest, "identifier is required")
		return
	}

	current := audio.GetAllAudio()
	if _, exists := current[identifier]; !exists {
		writeAPIError(w, http.StatusNotFound, "identifier not found: "+identifier)
		return
	}

	delete(current, identifier)

	if err := audio.SaveAudio(current); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, APIResponse[struct{}]{Success: true})
}
