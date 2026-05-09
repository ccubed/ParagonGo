package spells

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func SaveSpellSpec(spec *SpellData) error {
	if spec.SpellId == "" {
		return fmt.Errorf("cannot save spell with empty SpellId")
	}

	if err := spec.Validate(); err != nil {
		return err
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*SpellData](configs.GetFilePathsConfig().DataFiles.String()+`/spells`, spec, saveModes...); err != nil {
		return err
	}

	allSpells[spec.SpellId] = spec
	return nil
}

func DeleteSpellSpec(spellId string) error {
	spec := GetSpell(spellId)
	if spec == nil {
		return fmt.Errorf("spell %q not found", spellId)
	}

	basePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/spells/`)

	yamlPath := basePath + spec.Filepath()
	if err := os.Remove(yamlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing spell yaml: %w", err)
	}

	scriptPath := spec.GetScriptPath()
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing spell script: %w", err)
	}

	delete(allSpells, spellId)
	return nil
}

func SaveSpellScript(spellId string, content string) error {
	spec := GetSpell(spellId)
	if spec == nil {
		return fmt.Errorf("spell %q not found", spellId)
	}

	scriptPath := spec.GetScriptPath()

	if content == "" {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing spell script: %w", err)
		}
		return nil
	}

	return os.WriteFile(scriptPath, []byte(content), 0644)
}
