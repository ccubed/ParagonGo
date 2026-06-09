package items

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// GetAllAttackMessages returns a copy of the loaded attack message groups keyed
// by weapon subtype.
func GetAllAttackMessages() map[ItemSubType]*WeaponAttackMessageGroup {
	result := make(map[ItemSubType]*WeaponAttackMessageGroup, len(attackMessages))
	for k, v := range attackMessages {
		result[k] = v
	}
	return result
}

func AddAttackMessage(subtype ItemSubType, intensity Intensity, proximity, target, message string) error {
	group, ok := attackMessages[subtype]
	if !ok {
		return fmt.Errorf("unknown subtype: %s", subtype)
	}

	opts, ok := group.Options[intensity]
	if !ok {
		return fmt.Errorf("unknown intensity: %s", intensity)
	}

	msg := ItemMessage(message)

	switch proximity {
	case "together":
		switch target {
		case "toattacker":
			opts.Together.ToAttacker = append(opts.Together.ToAttacker, msg)
		case "todefender":
			opts.Together.ToDefender = append(opts.Together.ToDefender, msg)
		case "toroom":
			opts.Together.ToRoom = append(opts.Together.ToRoom, msg)
		default:
			return fmt.Errorf("unknown together target: %s", target)
		}
	case "separate":
		switch target {
		case "toattacker":
			opts.Separate.ToAttacker = append(opts.Separate.ToAttacker, msg)
		case "todefender":
			opts.Separate.ToDefender = append(opts.Separate.ToDefender, msg)
		case "toattackerroom":
			opts.Separate.ToAttackerRoom = append(opts.Separate.ToAttackerRoom, msg)
		case "todefenderroom":
			opts.Separate.ToDefenderRoom = append(opts.Separate.ToDefenderRoom, msg)
		default:
			return fmt.Errorf("unknown separate target: %s", target)
		}
	default:
		return fmt.Errorf("unknown proximity: %s (expected together or separate)", proximity)
	}

	group.Options[intensity] = opts

	return SaveAttackMessageGroup(group)
}

func UpdateAttackMessage(subtype ItemSubType, intensity Intensity, proximity, target string, index int, message string) error {
	group, ok := attackMessages[subtype]
	if !ok {
		return fmt.Errorf("unknown subtype: %s", subtype)
	}

	opts, ok := group.Options[intensity]
	if !ok {
		return fmt.Errorf("unknown intensity: %s", intensity)
	}

	update := func(sl MessageOptions, i int) (MessageOptions, error) {
		if i < 0 || i >= len(sl) {
			return nil, fmt.Errorf("index %d out of range (length %d)", i, len(sl))
		}
		sl[i] = ItemMessage(message)
		return sl, nil
	}

	var err error
	switch proximity {
	case "together":
		switch target {
		case "toattacker":
			opts.Together.ToAttacker, err = update(opts.Together.ToAttacker, index)
		case "todefender":
			opts.Together.ToDefender, err = update(opts.Together.ToDefender, index)
		case "toroom":
			opts.Together.ToRoom, err = update(opts.Together.ToRoom, index)
		default:
			return fmt.Errorf("unknown together target: %s", target)
		}
	case "separate":
		switch target {
		case "toattacker":
			opts.Separate.ToAttacker, err = update(opts.Separate.ToAttacker, index)
		case "todefender":
			opts.Separate.ToDefender, err = update(opts.Separate.ToDefender, index)
		case "toattackerroom":
			opts.Separate.ToAttackerRoom, err = update(opts.Separate.ToAttackerRoom, index)
		case "todefenderroom":
			opts.Separate.ToDefenderRoom, err = update(opts.Separate.ToDefenderRoom, index)
		default:
			return fmt.Errorf("unknown separate target: %s", target)
		}
	default:
		return fmt.Errorf("unknown proximity: %s (expected together or separate)", proximity)
	}

	if err != nil {
		return err
	}

	group.Options[intensity] = opts

	return SaveAttackMessageGroup(group)
}

func DeleteAttackMessage(subtype ItemSubType, intensity Intensity, proximity, target string, index int) error {
	group, ok := attackMessages[subtype]
	if !ok {
		return fmt.Errorf("unknown subtype: %s", subtype)
	}

	opts, ok := group.Options[intensity]
	if !ok {
		return fmt.Errorf("unknown intensity: %s", intensity)
	}

	remove := func(sl MessageOptions, i int) (MessageOptions, error) {
		if i < 0 || i >= len(sl) {
			return nil, fmt.Errorf("index %d out of range (length %d)", i, len(sl))
		}
		return append(sl[:i], sl[i+1:]...), nil
	}

	var err error
	switch proximity {
	case "together":
		switch target {
		case "toattacker":
			opts.Together.ToAttacker, err = remove(opts.Together.ToAttacker, index)
		case "todefender":
			opts.Together.ToDefender, err = remove(opts.Together.ToDefender, index)
		case "toroom":
			opts.Together.ToRoom, err = remove(opts.Together.ToRoom, index)
		default:
			return fmt.Errorf("unknown together target: %s", target)
		}
	case "separate":
		switch target {
		case "toattacker":
			opts.Separate.ToAttacker, err = remove(opts.Separate.ToAttacker, index)
		case "todefender":
			opts.Separate.ToDefender, err = remove(opts.Separate.ToDefender, index)
		case "toattackerroom":
			opts.Separate.ToAttackerRoom, err = remove(opts.Separate.ToAttackerRoom, index)
		case "todefenderroom":
			opts.Separate.ToDefenderRoom, err = remove(opts.Separate.ToDefenderRoom, index)
		default:
			return fmt.Errorf("unknown separate target: %s", target)
		}
	default:
		return fmt.Errorf("unknown proximity: %s (expected together or separate)", proximity)
	}

	if err != nil {
		return err
	}

	group.Options[intensity] = opts

	return SaveAttackMessageGroup(group)
}

func SaveAttackMessageGroup(group *WeaponAttackMessageGroup) error {
	basePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/combat-messages/`)
	filePath := basePath + group.Filepath()

	sectionComments := extractCombatComments(filePath)

	bytes, err := yaml.Marshal(group)
	if err != nil {
		return fmt.Errorf("marshaling attack messages: %w", err)
	}

	bytes = insertCombatComments(bytes, sectionComments)

	if err := util.WriteFile(filePath, bytes, 0644); err != nil {
		return fmt.Errorf("writing attack messages file: %w", err)
	}

	attackMessages[group.OptionId] = group
	return nil
}

func extractCombatComments(path string) map[string][]string {
	comments := make(map[string][]string)

	data, err := os.ReadFile(path)
	if err != nil {
		return comments
	}

	lines := strings.Split(string(data), "\n")
	var pending []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pending = append(pending, line)
			continue
		}

		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(line, ":") {
			key := strings.TrimSpace(line[:strings.Index(line, ":")])
			if len(pending) > 0 {
				comments[key] = append([]string{}, pending...)
				pending = nil
			}
		} else {
			pending = nil
		}
	}

	return comments
}

func insertCombatComments(data []byte, comments map[string][]string) []byte {
	if len(comments) == 0 {
		return data
	}

	lines := strings.Split(string(data), "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && trimmed != "" && strings.Contains(line, ":") {
			key := strings.TrimSpace(line[:strings.Index(line, ":")])
			if commentLines, ok := comments[key]; ok {
				result = append(result, commentLines...)
			}
		}
		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n"))
}

// SaveItemSpec persists an existing ItemSpec back to its data file and updates
// the in-memory cache.  The ItemId must already be set.
func SaveItemSpec(spec *ItemSpec) error {
	if spec.ItemId < 1 {
		return fmt.Errorf("cannot save item spec with invalid ItemId %d", spec.ItemId)
	}

	if err := spec.Validate(); err != nil {
		return err
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}

	if err := fileloader.SaveFlatFile[*ItemSpec](configs.GetFilePathsConfig().DataFiles.String()+`/items`, spec, saveModes...); err != nil {
		return err
	}

	items[spec.ItemId] = spec
	return nil
}

// DeleteItemSpec removes an item spec from disk (YAML + JS script) and from the
// in-memory cache.
func DeleteItemSpec(itemId int) error {
	spec := GetItemSpec(itemId)
	if spec == nil {
		return fmt.Errorf("item %d not found", itemId)
	}

	basePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/items/`)

	yamlPath := basePath + spec.Filepath()
	if err := os.Remove(yamlPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing item yaml: %w", err)
	}

	scriptPath := spec.GetScriptPath()
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing item script: %w", err)
	}

	delete(items, itemId)
	return nil
}

// SaveItemScript writes (or overwrites) the JavaScript file for an item.  If
// content is empty the script file is deleted instead.
func SaveItemScript(itemId int, content string, lang string) error {
	spec := GetItemSpec(itemId)
	if spec == nil {
		return fmt.Errorf("item %d not found", itemId)
	}

	scriptPath := spec.GetScriptPath()

	if content == "" {
		if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing item script: %w", err)
		}
		return nil
	}

	scriptPath = util.ApplyScriptLang(scriptPath, lang)
	return util.WriteFile(scriptPath, []byte(content), 0644)
}
