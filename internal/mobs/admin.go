package mobs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func GetAllMobSpecs() []Mob {
	result := make([]Mob, 0, len(mobs))
	for _, m := range mobs {
		result = append(result, *m)
	}
	return result
}

func SaveMobSpec(spec *Mob) error {
	if spec.MobId < 1 {
		return fmt.Errorf("cannot save mob spec with invalid MobId %d", spec.MobId)
	}

	if err := spec.Validate(); err != nil {
		return err
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*Mob](configs.GetFilePathsConfig().DataFiles.String()+`/mobs`, spec, saveModes...); err != nil {
		return err
	}

	oldName, hadOldName := mobNameCache[spec.MobId]

	mobs[spec.Id()] = spec
	mobNameCache[spec.MobId] = spec.Character.Name

	if hadOldName && oldName != spec.Character.Name {
		for i, n := range allMobNames {
			if n == oldName {
				allMobNames[i] = spec.Character.Name
				break
			}
		}
	} else if !hadOldName {
		allMobNames = append(allMobNames, spec.Character.Name)
	}

	return nil
}

func DeleteMobSpec(mobId MobId) error {
	spec := GetMobSpec(mobId)
	if spec == nil {
		return fmt.Errorf("mob %d not found", mobId)
	}

	basePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/mobs/`)

	yamlPath := basePath + spec.Filepath()
	if err := os.Remove(yamlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing mob yaml: %w", err)
	}

	scriptPath := spec.GetScriptPath()
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing mob script: %w", err)
	}

	name := mobNameCache[mobId]
	for i, n := range allMobNames {
		if n == name {
			allMobNames = append(allMobNames[:i], allMobNames[i+1:]...)
			break
		}
	}
	delete(mobNameCache, mobId)
	delete(mobs, int(mobId))

	return nil
}

func SaveMobScript(mobId MobId, content string, lang string) error {
	return SaveMobScriptForTag(mobId, "", content, lang)
}

func SaveMobScriptForTag(mobId MobId, tag string, content string, lang string) error {
	spec := GetMobSpec(mobId)
	if spec == nil {
		return fmt.Errorf("mob %d not found", mobId)
	}

	scriptPath := spec.GetScriptPathForTag(tag)

	if content == "" {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing mob script: %w", err)
		}
		return nil
	}

	scriptPath = util.ApplyScriptLang(scriptPath, lang)
	os.MkdirAll(filepath.Dir(scriptPath), os.ModePerm)
	return util.WriteFile(scriptPath, []byte(content), 0644)
}

func GetMobScriptForTag(mobId MobId, tag string) (string, error) {
	spec := GetMobSpec(mobId)
	if spec == nil {
		return "", fmt.Errorf("mob %d not found", mobId)
	}

	scriptPath := spec.GetScriptPathForTag(tag)
	if _, err := os.Stat(scriptPath); err != nil {
		return "", nil
	}
	bytes, err := util.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func StockMobShop(mobId MobId, entry characters.ShopItem) error {
	spec, ok := mobs[int(mobId)]
	if !ok {
		return fmt.Errorf("mob %d not found", mobId)
	}

	spec.Character.Shop = append(spec.Character.Shop, entry)

	return SaveMobSpec(spec)
}
