package buffs

import (
	"io/fs"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

var (
	pluginFileSystems []fileloader.ReadableGroupFS
	pluginScripts     = map[int]string{} // buffId -> JS source
)

// RegisterFS registers a plugin file system to be searched when loading buff
// data files. Must be called before LoadDataFiles().
func RegisterFS(f ...fileloader.ReadableGroupFS) {
	pluginFileSystems = append(pluginFileSystems, f...)
}

// RegisterBuffScript registers an embedded JS script for a given buff ID.
// This is used by modules that embed their scripts rather than placing them
// on disk alongside the YAML definition.
func RegisterBuffScript(buffId int, script string) {
	if buffId < 0 {
		buffId *= -1
	}
	pluginScripts[buffId] = script
}

// getPluginScript returns the registered plugin script for buffId, or "".
func getPluginScript(buffId int) string {
	if buffId < 0 {
		buffId *= -1
	}
	return pluginScripts[buffId]
}

// loadPluginBuffs walks every sub-filesystem of every registered plugin FS,
// reading buff YAML files from a "buffs/" prefix and merging them into dst.
// Disk-loaded buffs take precedence: a plugin buff with a duplicate id is
// logged and skipped.
func loadPluginBuffs(dst map[int]*BuffSpec) {
	for _, groupFS := range pluginFileSystems {
		for subFS := range groupFS.AllFileSubSystems {
			loadBuffsFromFS(subFS, dst)
		}
	}
}

func loadBuffsFromFS(subFS fs.ReadFileFS, dst map[int]*BuffSpec) {
	if pl, ok := subFS.(fileloader.PathLister); ok {
		for _, path := range pl.KnownPaths() {
			if !strings.HasPrefix(path, `buffs/`) || !strings.HasSuffix(path, `.yaml`) {
				continue
			}
			loadBuffFileFromFS(subFS, path, dst)
		}
		return
	}

	_ = fs.WalkDir(subFS, `buffs`, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, `.yaml`) {
			return nil
		}
		loadBuffFileFromFS(subFS, path, dst)
		return nil
	})
}

func loadBuffFileFromFS(subFS fs.ReadFileFS, path string, dst map[int]*BuffSpec) {
	b, err := subFS.ReadFile(path)
	if err != nil {
		mudlog.Error("buffs.loadBuffsFromFS", "path", path, "error", err)
		return
	}

	var spec BuffSpec
	if err := yaml.Unmarshal(b, &spec); err != nil {
		mudlog.Error("buffs.loadBuffsFromFS", "path", path, "error", err)
		return
	}

	if !strings.HasSuffix(path, spec.Filepath()) {
		mudlog.Error("buffs.loadBuffsFromFS", "path", path, "expected suffix", spec.Filepath(), "error", "filepath mismatch")
		return
	}

	if err := spec.Validate(); err != nil {
		mudlog.Error("buffs.loadBuffsFromFS", "path", path, "error", err)
		return
	}

	if _, exists := dst[spec.BuffId]; exists {
		mudlog.Error("buffs.loadBuffsFromFS", "buffId", spec.BuffId, "path", path, "error", "duplicate buff id")
		return
	}

	dst[spec.BuffId] = &spec
}
