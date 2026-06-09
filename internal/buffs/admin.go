package buffs

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GetAllBuffSpecs returns a copy of all loaded buff specs keyed by buffId.
func GetAllBuffSpecs() map[int]*BuffSpec {
	result := make(map[int]*BuffSpec, len(buffs))
	for k, v := range buffs {
		result[k] = v
	}
	return result
}

// SaveBuffSpec persists a BuffSpec to its data file and updates the in-memory
// cache.  The BuffId must already be set.
func SaveBuffSpec(spec *BuffSpec) error {
	if spec.BuffId < 0 {
		return fmt.Errorf("cannot save buff spec with invalid BuffId %d", spec.BuffId)
	}

	if err := spec.Validate(); err != nil {
		return err
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*BuffSpec](configs.GetFilePathsConfig().DataFiles.String()+`/buffs`, spec, saveModes...); err != nil {
		return err
	}

	buffs[spec.BuffId] = spec
	return nil
}

// DeleteBuffSpec removes a buff spec from disk (YAML + JS script) and from the
// in-memory cache.
func DeleteBuffSpec(buffId int) error {
	spec := GetBuffSpec(buffId)
	if spec == nil {
		return fmt.Errorf("buff %d not found", buffId)
	}

	basePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/buffs/`)

	yamlPath := basePath + spec.Filepath()
	if err := os.Remove(yamlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing buff yaml: %w", err)
	}

	scriptPath := spec.GetScriptPath()
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing buff script: %w", err)
	}

	delete(buffs, buffId)
	return nil
}

// SaveBuffScript writes (or overwrites) the JavaScript file for a buff.  If
// content is empty the script file is deleted instead.
func SaveBuffScript(buffId int, content string, lang string) error {
	spec := GetBuffSpec(buffId)
	if spec == nil {
		return fmt.Errorf("buff %d not found", buffId)
	}

	scriptPath := spec.GetScriptPath()

	if content == "" {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing buff script: %w", err)
		}
		return nil
	}

	scriptPath = util.ApplyScriptLang(scriptPath, lang)
	return util.WriteFile(scriptPath, []byte(content), 0644)
}
