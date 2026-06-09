package buffs

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
	pluginScripts = map[int]string{}
}

func TestLoadPluginBuffs_MergesValidBuff(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	yamlData := []byte("buffid: 9001\nname: Plugin Buff\ntriggerrate: 1 round\ntriggercount: 3\n")
	RegisterFS(newFakeFS(map[string][]byte{
		`buffs/9001-plugin_buff.yaml`: yamlData,
	}))

	dst := map[int]*BuffSpec{}
	loadPluginBuffs(dst)

	b, ok := dst[9001]
	if !ok {
		t.Fatalf("expected buff 9001 to be loaded")
	}
	if b.Name != "Plugin Buff" {
		t.Fatalf("expected name %q, got %q", "Plugin Buff", b.Name)
	}
}

func TestLoadPluginBuffs_IgnoresWrongPrefix(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`items/9002-x.yaml`: []byte("buffid: 9002\nname: X\ntriggerrate: 1 round\ntriggercount: 1\n"),
	}))

	dst := map[int]*BuffSpec{}
	loadPluginBuffs(dst)

	if len(dst) != 0 {
		t.Fatalf("expected wrong-prefix buff to be ignored, got %d", len(dst))
	}
}

func TestLoadPluginBuffs_RejectsFilepathMismatch(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`buffs/wrong.yaml`: []byte("buffid: 9003\nname: Good Buff\ntriggerrate: 1 round\ntriggercount: 1\n"),
	}))

	dst := map[int]*BuffSpec{}
	loadPluginBuffs(dst)

	if len(dst) != 0 {
		t.Fatalf("expected mismatched-filepath buff to be skipped, got %d", len(dst))
	}
}

func TestLoadPluginBuffs_SkipsInvalid(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	// triggercount of 0 fails Validate().
	RegisterFS(newFakeFS(map[string][]byte{
		`buffs/9004-bad.yaml`: []byte("buffid: 9004\nname: Bad\ntriggerrate: 1 round\ntriggercount: 0\n"),
	}))

	dst := map[int]*BuffSpec{}
	loadPluginBuffs(dst)

	if len(dst) != 0 {
		t.Fatalf("expected invalid buff to be skipped, got %d", len(dst))
	}
}

func TestLoadPluginBuffs_DiskWinsOnDuplicate(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`buffs/9005-plugin.yaml`: []byte("buffid: 9005\nname: Plugin\ntriggerrate: 1 round\ntriggercount: 1\n"),
	}))

	dst := map[int]*BuffSpec{
		9005: {BuffId: 9005, Name: "Disk"},
	}
	loadPluginBuffs(dst)

	if dst[9005].Name != "Disk" {
		t.Fatalf("expected disk buff to win, got %q", dst[9005].Name)
	}
}

func TestRegisterBuffScript_ReturnedByGetScript(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterBuffScript(9006, "onApply()")

	spec := &BuffSpec{BuffId: 9006, Name: "Scripted"}
	if got := spec.GetScript(); got != "onApply()" {
		t.Fatalf("expected plugin script, got %q", got)
	}

	// Negative ids resolve to the same registered script.
	spec.BuffId = -9006
	if got := spec.GetScript(); got != "onApply()" {
		t.Fatalf("expected plugin script for negative id, got %q", got)
	}
}
