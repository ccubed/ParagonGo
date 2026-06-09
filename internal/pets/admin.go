package pets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GetAllPetSpecs returns a copy of every loaded pet type keyed by type ID.
func GetAllPetSpecs() map[string]Pet {
	result := make(map[string]Pet, len(petTypes))
	for k, v := range petTypes {
		result[k] = *v
	}
	return result
}

// SavePetSpec validates, persists a pet spec to disk, and updates the in-memory cache.
func SavePetSpec(p *Pet) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Type == "" {
		return errors.New("pet type is required")
	}
	p.Type = strings.ToLower(strings.TrimSpace(p.Type))

	if err := p.Save(); err != nil {
		return err
	}
	cp := *p
	petTypes[p.Type] = &cp
	return nil
}

// DeletePetSpec removes a pet type from disk and the in-memory cache.
func DeletePetSpec(petType string) error {
	petType = strings.ToLower(strings.TrimSpace(petType))
	p, ok := petTypes[petType]
	if !ok {
		return fmt.Errorf("pet type %q not found", petType)
	}
	path := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `pets`, `/`, p.Filename())
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing pet file: %w", err)
	}
	scriptPath := p.GetScriptPath()
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing pet script: %w", err)
	}
	delete(petTypes, petType)
	return nil
}

// CreatePetSpec registers a new pet type. Returns an error if the type already exists.
func CreatePetSpec(p *Pet) error {
	if p.Type == "" {
		return errors.New("pet type is required")
	}
	p.Type = strings.ToLower(strings.TrimSpace(p.Type))
	if _, exists := petTypes[p.Type]; exists {
		return fmt.Errorf("pet type %q already exists", p.Type)
	}
	return SavePetSpec(p)
}

// SavePetScript writes (or removes) the JavaScript script file for a pet type.
func SavePetScript(petType string, content string, lang string) error {
	petType = strings.ToLower(strings.TrimSpace(petType))
	p, ok := petTypes[petType]
	if !ok {
		return fmt.Errorf("pet type %q not found", petType)
	}

	scriptPath := p.GetScriptPath()

	if content == "" {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing pet script: %w", err)
		}
		return nil
	}

	scriptPath = util.ApplyScriptLang(scriptPath, lang)
	if err := os.MkdirAll(filepath.Dir(scriptPath), os.ModePerm); err != nil {
		return fmt.Errorf("creating pet scripts directory: %w", err)
	}
	return util.WriteFile(scriptPath, []byte(content), 0644)
}
