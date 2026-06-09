package pets

import (
	"io/fs"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

var (
	pluginFileSystems []fileloader.ReadableGroupFS
	pluginScripts     = map[string]string{} // pet type -> JS source
)

// RegisterFS registers a plugin file system to be searched when loading pet
// data files. Must be called before LoadDataFiles().
func RegisterFS(f ...fileloader.ReadableGroupFS) {
	pluginFileSystems = append(pluginFileSystems, f...)
}

// RegisterPetScript registers an embedded JS script for a given pet type.
// This is used by modules that embed their scripts rather than placing them
// on disk alongside the YAML definition.
func RegisterPetScript(petType string, script string) {
	pluginScripts[petType] = script
}

// getPluginScript returns the registered plugin script for petType, or "".
func getPluginScript(petType string) string {
	return pluginScripts[petType]
}

// loadPluginPets walks every sub-filesystem of every registered plugin FS,
// reading pet YAML files from a "pets/" prefix and merging them into dst.
// Disk-loaded pets take precedence: a plugin pet with a duplicate type is
// logged and skipped.
func loadPluginPets(dst map[string]*Pet) {
	for _, groupFS := range pluginFileSystems {
		for subFS := range groupFS.AllFileSubSystems {
			loadPetsFromFS(subFS, dst)
		}
	}
}

func loadPetsFromFS(subFS fs.ReadFileFS, dst map[string]*Pet) {
	if pl, ok := subFS.(fileloader.PathLister); ok {
		for _, path := range pl.KnownPaths() {
			if !strings.HasPrefix(path, `pets/`) || !strings.HasSuffix(path, `.yaml`) {
				continue
			}
			loadPetFileFromFS(subFS, path, dst)
		}
		return
	}

	_ = fs.WalkDir(subFS, `pets`, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, `.yaml`) {
			return nil
		}
		loadPetFileFromFS(subFS, path, dst)
		return nil
	})
}

func loadPetFileFromFS(subFS fs.ReadFileFS, path string, dst map[string]*Pet) {
	b, err := subFS.ReadFile(path)
	if err != nil {
		mudlog.Error("pets.loadPetsFromFS", "path", path, "error", err)
		return
	}

	var pet Pet
	if err := yaml.Unmarshal(b, &pet); err != nil {
		mudlog.Error("pets.loadPetsFromFS", "path", path, "error", err)
		return
	}

	if !strings.HasSuffix(path, pet.Filepath()) {
		mudlog.Error("pets.loadPetsFromFS", "path", path, "expected suffix", pet.Filepath(), "error", "filepath mismatch")
		return
	}

	if err := pet.Validate(); err != nil {
		mudlog.Error("pets.loadPetsFromFS", "path", path, "error", err)
		return
	}

	if _, exists := dst[pet.Id()]; exists {
		mudlog.Error("pets.loadPetsFromFS", "petType", pet.Id(), "path", path, "error", "duplicate pet type")
		return
	}

	dst[pet.Id()] = &pet
}
