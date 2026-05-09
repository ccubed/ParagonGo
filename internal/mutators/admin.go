package mutators

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var validMutatorIdPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

func ValidateMutatorId(id string) error {
	if id == "" {
		return fmt.Errorf("mutatorid is required")
	}
	if !validMutatorIdPattern.MatchString(id) {
		return fmt.Errorf("mutatorid must be lowercase alphanumeric with hyphens/underscores, starting with a letter: %q", id)
	}
	return nil
}

func ValidateBuffIds(ids []int) error {
	for _, id := range ids {
		if buffs.GetBuffSpec(id) == nil {
			return fmt.Errorf("invalid buff id: %d", id)
		}
	}
	return nil
}

func ValidateSpec(spec *MutatorSpec) error {
	if err := ValidateMutatorId(spec.MutatorId); err != nil {
		return err
	}

	if spec.LightMod < -2 || spec.LightMod > 2 {
		return fmt.Errorf("lightmod must be between -2 and 2, got %d", spec.LightMod)
	}

	if err := ValidateBuffIds(spec.PlayerBuffIds); err != nil {
		return fmt.Errorf("playerbuffids: %w", err)
	}
	if err := ValidateBuffIds(spec.MobBuffIds); err != nil {
		return fmt.Errorf("mobbuffids: %w", err)
	}
	if err := ValidateBuffIds(spec.NativeBuffIds); err != nil {
		return fmt.Errorf("nativebuffids: %w", err)
	}

	if spec.DecayIntoId != "" {
		if err := ValidateMutatorId(spec.DecayIntoId); err != nil {
			return fmt.Errorf("decayintoid: %w", err)
		}
	}

	if spec.NameModifier != nil && spec.NameModifier.Behavior != "" && !spec.NameModifier.Behavior.IsValid() {
		return fmt.Errorf("namemodifier behavior is invalid: %q", spec.NameModifier.Behavior)
	}
	if spec.DescriptionModifier != nil && spec.DescriptionModifier.Behavior != "" && !spec.DescriptionModifier.Behavior.IsValid() {
		return fmt.Errorf("descriptionmodifier behavior is invalid: %q", spec.DescriptionModifier.Behavior)
	}
	if spec.AlertModifier != nil && spec.AlertModifier.Behavior != "" && !spec.AlertModifier.Behavior.IsValid() {
		return fmt.Errorf("alertmodifier behavior is invalid: %q", spec.AlertModifier.Behavior)
	}

	if spec.Pvp.Enabled && spec.Pvp.Disabled {
		return fmt.Errorf("pvp cannot be both enabled and disabled")
	}

	return spec.Validate()
}

func CreateNewMutatorSpec(spec MutatorSpec) (string, error) {
	spec.MutatorId = strings.ToLower(strings.TrimSpace(spec.MutatorId))

	if err := ValidateSpec(&spec); err != nil {
		return "", err
	}

	if _, exists := allMutators[spec.MutatorId]; exists {
		return "", fmt.Errorf("mutator already exists: %s", spec.MutatorId)
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*MutatorSpec](configs.GetFilePathsConfig().DataFiles.String()+`/mutators`, &spec, saveModes...); err != nil {
		return "", err
	}

	allMutators[spec.MutatorId] = &spec
	return spec.MutatorId, nil
}

func SaveMutatorSpec(spec *MutatorSpec) error {
	if spec.MutatorId == "" {
		return fmt.Errorf("cannot save mutator spec with empty mutatorid")
	}

	if err := ValidateSpec(spec); err != nil {
		return err
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*MutatorSpec](configs.GetFilePathsConfig().DataFiles.String()+`/mutators`, spec, saveModes...); err != nil {
		return err
	}

	allMutators[spec.MutatorId] = spec
	return nil
}

func DeleteMutatorSpec(mutatorId string) error {
	spec := GetMutatorSpec(mutatorId)
	if spec == nil {
		return fmt.Errorf("mutator not found: %s", mutatorId)
	}

	basePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/mutators/`)

	yamlPath := basePath + spec.Filepath()
	if err := os.Remove(yamlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing mutator yaml: %w", err)
	}

	delete(allMutators, mutatorId)
	return nil
}
