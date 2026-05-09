package colorpatterns

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// GetAllColorPatterns returns a copy of all loaded color patterns (name → ANSI codes).
func GetAllColorPatterns() map[string][]int {
	result := make(map[string][]int, len(numericPatterns))
	for k, v := range numericPatterns {
		cp := make([]int, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// SaveColorPattern adds or replaces a named color pattern in memory and persists the file.
func SaveColorPattern(name string, colors []int) error {
	if name == "" {
		return fmt.Errorf("color pattern name cannot be empty")
	}
	if len(colors) == 0 {
		return fmt.Errorf("color pattern must have at least one color value")
	}
	cp := make([]int, len(colors))
	copy(cp, colors)
	numericPatterns[name] = cp
	colorsCompiled = false
	CompileColorPatterns()
	return saveColorPatternsFile()
}

// DeleteColorPattern removes a named color pattern from memory and persists the file.
func DeleteColorPattern(name string) error {
	if _, ok := numericPatterns[name]; !ok {
		return fmt.Errorf("color pattern %q not found", name)
	}
	delete(numericPatterns, name)
	delete(ShortTagPatterns, name)
	return saveColorPatternsFile()
}

func saveColorPatternsFile() error {
	path := util.FilePath(configs.GetFilePathsConfig().DataFiles.String() + `/color-patterns.yaml`)
	bytes, err := yaml.Marshal(numericPatterns)
	if err != nil {
		return fmt.Errorf("marshaling color patterns: %w", err)
	}
	return os.WriteFile(path, bytes, 0644)
}
