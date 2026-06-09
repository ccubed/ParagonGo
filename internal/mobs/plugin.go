package mobs

import (
	"io/fs"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// mobScriptKey identifies an embedded mob script by mob id and script tag.
// An empty tag refers to the mob's base (untagged) script.
type mobScriptKey struct {
	mobId int
	tag   string
}

var (
	pluginFileSystems []fileloader.ReadableGroupFS
	pluginScripts     = map[mobScriptKey]string{} // (mobId, tag) -> JS source
)

// RegisterFS registers a plugin file system to be searched when loading mob
// data files. Must be called before LoadDataFiles().
func RegisterFS(f ...fileloader.ReadableGroupFS) {
	pluginFileSystems = append(pluginFileSystems, f...)
}

// RegisterMobScript registers an embedded JS script for a given mob ID and
// script tag (empty tag = base script). This is used by modules that embed
// their scripts rather than placing them on disk alongside the YAML definition.
//
// Note: embedded mob scripts are not enumerated by GetAllScriptTags(), so
// module-provided mob scripts are not editable through the admin script-tag UI.
func RegisterMobScript(mobId int, tag string, script string) {
	pluginScripts[mobScriptKey{mobId: mobId, tag: tag}] = script
}

// getPluginScript returns the registered plugin script for (mobId, tag), or "".
func getPluginScript(mobId int, tag string) string {
	return pluginScripts[mobScriptKey{mobId: mobId, tag: tag}]
}

// loadPluginMobs walks every sub-filesystem of every registered plugin FS,
// reading mob YAML files from a "mobs/" prefix and merging them into dst.
// Disk-loaded mobs take precedence: a plugin mob with a duplicate id is logged
// and skipped.
//
// Mob spec files live under "mobs/<zone>/<mobId>-<name>.yaml". Files under a
// "scripts/" segment are ignored so script-adjacent yaml is not mistaken for a
// mob spec.
func loadPluginMobs(dst map[int]*Mob) {
	for _, groupFS := range pluginFileSystems {
		for subFS := range groupFS.AllFileSubSystems {
			loadMobsFromFS(subFS, dst)
		}
	}
}

func loadMobsFromFS(subFS fs.ReadFileFS, dst map[int]*Mob) {
	if pl, ok := subFS.(fileloader.PathLister); ok {
		for _, path := range pl.KnownPaths() {
			if !isMobSpecPath(path) {
				continue
			}
			loadMobFileFromFS(subFS, path, dst)
		}
		return
	}

	_ = fs.WalkDir(subFS, `mobs`, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isMobSpecPath(path) {
			return nil
		}
		loadMobFileFromFS(subFS, path, dst)
		return nil
	})
}

func isMobSpecPath(path string) bool {
	if !strings.HasPrefix(path, `mobs/`) || !strings.HasSuffix(path, `.yaml`) {
		return false
	}
	if strings.Contains(path, `/scripts/`) {
		return false
	}
	return true
}

func loadMobFileFromFS(subFS fs.ReadFileFS, path string, dst map[int]*Mob) {
	b, err := subFS.ReadFile(path)
	if err != nil {
		mudlog.Error("mobs.loadMobsFromFS", "path", path, "error", err)
		return
	}

	var mob Mob
	if err := yaml.Unmarshal(b, &mob); err != nil {
		mudlog.Error("mobs.loadMobsFromFS", "path", path, "error", err)
		return
	}

	// During load mobNameCache is not yet populated for this mob, so Filepath()
	// derives the filename from Character.Name. The embedded file must be named
	// "<zone>/<mobId>-<filename-from-character-name>.yaml" to satisfy this.
	if !strings.HasSuffix(path, mob.Filepath()) {
		mudlog.Error("mobs.loadMobsFromFS", "path", path, "expected suffix", mob.Filepath(), "error", "filepath mismatch")
		return
	}

	if err := mob.Validate(); err != nil {
		mudlog.Error("mobs.loadMobsFromFS", "path", path, "error", err)
		return
	}

	if _, exists := dst[mob.Id()]; exists {
		mudlog.Error("mobs.loadMobsFromFS", "mobId", mob.Id(), "path", path, "error", "duplicate mob id")
		return
	}

	dst[mob.Id()] = &mob
}
