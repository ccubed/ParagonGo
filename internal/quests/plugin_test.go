package quests

import (
	"io/fs"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// fakeFS is an in-memory ReadableGroupFS + PathLister used to exercise the
// plugin loader without touching disk. It acts as a single sub-filesystem that
// is also its own group.
type fakeFS struct {
	files map[string][]byte
}

func newFakeFS(files map[string][]byte) *fakeFS {
	return &fakeFS{files: files}
}

func (f *fakeFS) ReadFile(name string) ([]byte, error) {
	if b, ok := f.files[name]; ok {
		return b, nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (f *fakeFS) KnownPaths() []string {
	paths := make([]string, 0, len(f.files))
	for p := range f.files {
		paths = append(paths, p)
	}
	return paths
}

func (f *fakeFS) AllFileSubSystems(yield func(fs.ReadFileFS) bool) {
	yield(f)
}

// resetPluginState clears package-level plugin registration between tests.
func resetPluginState() {
	pluginFileSystems = nil
}

func TestLoadPluginQuests_MergesValidQuest(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	yamlData := []byte("questid: 9001\nname: Plugin Quest\nsteps:\n  - id: start\n    description: Begin\n")
	RegisterFS(newFakeFS(map[string][]byte{
		`quests/9001-plugin_quest.yaml`: yamlData,
	}))

	dst := map[int]*Quest{}
	loadPluginQuests(dst)

	q, ok := dst[9001]
	if !ok {
		t.Fatalf("expected quest 9001 to be loaded")
	}
	if q.Name != "Plugin Quest" {
		t.Fatalf("expected name %q, got %q", "Plugin Quest", q.Name)
	}
}

func TestLoadPluginQuests_IgnoresWrongPrefixAndExtension(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`items/9002-not-a-quest.yaml`: []byte("questid: 9002\nname: X\n"),
		`quests/9003-readme.txt`:      []byte("not yaml"),
		`quests/9004-good.yaml`:       []byte("questid: 9004\nname: Good\n"),
	}))

	dst := map[int]*Quest{}
	loadPluginQuests(dst)

	if _, ok := dst[9002]; ok {
		t.Fatalf("wrong-prefix quest should be ignored")
	}
	if _, ok := dst[9004]; !ok {
		t.Fatalf("expected quest 9004 to be loaded")
	}
	if len(dst) != 1 {
		t.Fatalf("expected exactly 1 quest, got %d", len(dst))
	}
}

func TestLoadPluginQuests_RejectsFilepathMismatch(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	// questid 9005 with name "Good" should be at quests/9005-good.yaml;
	// here the filename does not match Filepath().
	RegisterFS(newFakeFS(map[string][]byte{
		`quests/wrong-name.yaml`: []byte("questid: 9005\nname: Good\n"),
	}))

	dst := map[int]*Quest{}
	loadPluginQuests(dst)

	if len(dst) != 0 {
		t.Fatalf("expected mismatched-filepath quest to be skipped, got %d", len(dst))
	}
}

func TestLoadPluginQuests_SkipsInvalid(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	// Empty name fails Validate(). Filepath for an empty name is "9006-.yaml".
	RegisterFS(newFakeFS(map[string][]byte{
		`quests/9006-.yaml`: []byte("questid: 9006\nname: \"\"\n"),
	}))

	dst := map[int]*Quest{}
	loadPluginQuests(dst)

	if len(dst) != 0 {
		t.Fatalf("expected invalid quest to be skipped, got %d", len(dst))
	}
}

func TestLoadPluginQuests_DiskWinsOnDuplicate(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`quests/9007-plugin.yaml`: []byte("questid: 9007\nname: Plugin\n"),
	}))

	dst := map[int]*Quest{
		9007: {QuestId: 9007, Name: "Disk"},
	}
	loadPluginQuests(dst)

	if dst[9007].Name != "Disk" {
		t.Fatalf("expected disk quest to win, got %q", dst[9007].Name)
	}
}
