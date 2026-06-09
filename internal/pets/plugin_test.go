package pets

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
	pluginScripts = map[string]string{}
}

func TestLoadPluginPets_MergesValidPet(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	yamlData := []byte("type: pluginwolf\n")
	RegisterFS(newFakeFS(map[string][]byte{
		`pets/pluginwolf.yaml`: yamlData,
	}))

	dst := map[string]*Pet{}
	loadPluginPets(dst)

	p, ok := dst["pluginwolf"]
	if !ok {
		t.Fatalf("expected pet pluginwolf to be loaded")
	}
	if p.Type != "pluginwolf" {
		t.Fatalf("expected type %q, got %q", "pluginwolf", p.Type)
	}
}

func TestLoadPluginPets_IgnoresWrongPrefix(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`mobs/pluginwolf.yaml`: []byte("type: pluginwolf\n"),
	}))

	dst := map[string]*Pet{}
	loadPluginPets(dst)

	if len(dst) != 0 {
		t.Fatalf("expected wrong-prefix pet to be ignored, got %d", len(dst))
	}
}

func TestLoadPluginPets_RejectsFilepathMismatch(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`pets/wrong.yaml`: []byte("type: pluginwolf\n"),
	}))

	dst := map[string]*Pet{}
	loadPluginPets(dst)

	if len(dst) != 0 {
		t.Fatalf("expected mismatched-filepath pet to be skipped, got %d", len(dst))
	}
}

func TestLoadPluginPets_DiskWinsOnDuplicate(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`pets/pluginwolf.yaml`: []byte("type: pluginwolf\nnamestyle: \":plugin\"\n"),
	}))

	dst := map[string]*Pet{
		"pluginwolf": {Type: "pluginwolf", NameStyle: ":disk"},
	}
	loadPluginPets(dst)

	if dst["pluginwolf"].NameStyle != ":disk" {
		t.Fatalf("expected disk pet to win, got %q", dst["pluginwolf"].NameStyle)
	}
}

func TestRegisterPetScript_ReturnedByGetScript(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterPetScript("pluginwolf", "onAct()")

	p := &Pet{Type: "pluginwolf"}
	if got := p.GetScript(); got != "onAct()" {
		t.Fatalf("expected plugin script, got %q", got)
	}
	if !p.HasScript() {
		t.Fatalf("expected HasScript to be true for plugin script")
	}
}
