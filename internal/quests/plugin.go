package quests

import (
	"io/fs"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

var (
	pluginFileSystems []fileloader.ReadableGroupFS
)

// RegisterFS registers a plugin file system to be searched when loading quest
// data files. Must be called before LoadDataFiles().
func RegisterFS(f ...fileloader.ReadableGroupFS) {
	pluginFileSystems = append(pluginFileSystems, f...)
}

// loadPluginQuests walks every sub-filesystem of every registered plugin FS,
// reading quest YAML files from a "quests/" prefix and merging them into dst.
// Disk-loaded quests take precedence: a plugin quest with a duplicate id is
// logged and skipped.
func loadPluginQuests(dst map[int]*Quest) {
	for _, groupFS := range pluginFileSystems {
		for subFS := range groupFS.AllFileSubSystems {
			loadQuestsFromFS(subFS, dst)
		}
	}
}

func loadQuestsFromFS(subFS fs.ReadFileFS, dst map[int]*Quest) {
	// PluginFiles does not support directory traversal, so use KnownPaths
	// when available to enumerate files directly.
	if pl, ok := subFS.(fileloader.PathLister); ok {
		for _, path := range pl.KnownPaths() {
			if !strings.HasPrefix(path, `quests/`) || !strings.HasSuffix(path, `.yaml`) {
				continue
			}
			loadQuestFileFromFS(subFS, path, dst)
		}
		return
	}

	// Fallback: standard directory walk for FSes that support it.
	_ = fs.WalkDir(subFS, `quests`, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, `.yaml`) {
			return nil
		}
		loadQuestFileFromFS(subFS, path, dst)
		return nil
	})
}

func loadQuestFileFromFS(subFS fs.ReadFileFS, path string, dst map[int]*Quest) {
	b, err := subFS.ReadFile(path)
	if err != nil {
		mudlog.Error("quests.loadQuestsFromFS", "path", path, "error", err)
		return
	}

	var quest Quest
	if err := yaml.Unmarshal(b, &quest); err != nil {
		mudlog.Error("quests.loadQuestsFromFS", "path", path, "error", err)
		return
	}

	// Validate the Filepath() claim matches the actual path so the same
	// rules as the disk loader apply.
	if !strings.HasSuffix(path, quest.Filepath()) {
		mudlog.Error("quests.loadQuestsFromFS", "path", path, "expected suffix", quest.Filepath(), "error", "filepath mismatch")
		return
	}

	if err := quest.Validate(); err != nil {
		mudlog.Error("quests.loadQuestsFromFS", "path", path, "error", err)
		return
	}

	if _, exists := dst[quest.QuestId]; exists {
		mudlog.Error("quests.loadQuestsFromFS", "questId", quest.QuestId, "path", path, "error", "duplicate quest id")
		return
	}

	dst[quest.QuestId] = &quest
}
