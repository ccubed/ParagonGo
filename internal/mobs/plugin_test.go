package mobs

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

func (f *fakeFS) Open(name string) (fs.File, error) { return nil, fs.ErrNotExist }

func (f *fakeFS) KnownPaths() []string {
	paths := make([]string, 0, len(f.files))
	for p := range f.files {
		paths = append(paths, p)
	}
	return paths
}

func (f *fakeFS) AllFileSubSystems(yield func(fs.ReadFileFS) bool) { yield(f) }

func resetPluginState() {
	pluginFileSystems = nil
	pluginScripts = map[mobScriptKey]string{}
}

func TestIsMobSpecPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{`mobs/testzone/9001-goblin.yaml`, true},
		{`mobs/9001-goblin.yaml`, true},
		{`mobs/testzone/scripts/9001-goblin.js`, false},
		{`mobs/testzone/scripts/9001-goblin.yaml`, false},
		{`items/9001-goblin.yaml`, false},
		{`mobs/testzone/9001-goblin.js`, false},
	}
	for _, c := range cases {
		if got := isMobSpecPath(c.path); got != c.want {
			t.Errorf("isMobSpecPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestLoadPluginMobs_MergesValidMob(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	// zone "testzone", mobId 9001, Character.Name "Plugin Goblin" =>
	// Filepath() = testzone/9001-plugin_goblin.yaml
	yamlData := []byte("mobid: 9001\nzone: testzone\ncharacter:\n  name: Plugin Goblin\n")
	RegisterFS(newFakeFS(map[string][]byte{
		`mobs/testzone/9001-plugin_goblin.yaml`: yamlData,
		// A script-adjacent yaml that must be ignored.
		`mobs/testzone/scripts/9001-plugin_goblin.yaml`: []byte("mobid: 9999\n"),
	}))

	dst := map[int]*Mob{}
	loadPluginMobs(dst)

	mob, ok := dst[9001]
	if !ok {
		t.Fatalf("expected mob 9001 to be loaded")
	}
	if mob.Character.Name != "Plugin Goblin" {
		t.Fatalf("expected name %q, got %q", "Plugin Goblin", mob.Character.Name)
	}
	if _, ok := dst[9999]; ok {
		t.Fatalf("script-adjacent yaml should have been ignored")
	}
}

func TestLoadPluginMobs_RejectsFilepathMismatch(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`mobs/testzone/wrong-name.yaml`: []byte("mobid: 9002\nzone: testzone\ncharacter:\n  name: Plugin Goblin\n"),
	}))

	dst := map[int]*Mob{}
	loadPluginMobs(dst)

	if len(dst) != 0 {
		t.Fatalf("expected mismatched-filepath mob to be skipped, got %d", len(dst))
	}
}

func TestLoadPluginMobs_DiskWinsOnDuplicate(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`mobs/testzone/9003-plugin_goblin.yaml`: []byte("mobid: 9003\nzone: testzone\ncharacter:\n  name: Plugin Goblin\n"),
	}))

	existing := &Mob{MobId: 9003}
	existing.Character.Name = "Disk Goblin"
	dst := map[int]*Mob{9003: existing}

	loadPluginMobs(dst)

	if dst[9003].Character.Name != "Disk Goblin" {
		t.Fatalf("expected disk mob to win, got %q", dst[9003].Character.Name)
	}
}

func TestRegisterMobScript_ReturnedByGetScript(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterMobScript(9004, "", "onIdle()")
	RegisterMobScript(9004, "combat", "onCombat()")

	base := &Mob{MobId: 9004}
	if got := base.GetScript(); got != "onIdle()" {
		t.Fatalf("expected base plugin script, got %q", got)
	}
	if !base.HasScript() {
		t.Fatalf("expected HasScript true for base plugin script")
	}

	tagged := &Mob{MobId: 9004, ScriptTag: "combat"}
	if got := tagged.GetScript(); got != "onCombat()" {
		t.Fatalf("expected tagged plugin script, got %q", got)
	}
}
