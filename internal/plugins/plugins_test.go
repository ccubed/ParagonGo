package plugins

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFS embeds the fixture tree used by all tests in this file.
// The tree contains:
//
//	testdata/datafiles/html/public/index.html  -> short key: html/public/index.html
//	testdata/datafiles/items/test-sword.yaml   -> short key: items/test-sword.yaml
//	testdata/data-overlays/config.yaml         -> short key: data-overlays/config.yaml
//
//go:embed testdata
var testFS embed.FS

// newTestPlugin returns a Plugin with testFS attached, constructed directly
// to avoid touching the package-level registry.
func newTestPlugin(t *testing.T) *Plugin {
	t.Helper()
	p := &Plugin{name: "testplugin", version: "1.0"}
	require.NoError(t, p.AttachFileSystem(testFS))
	return p
}

// withRegistryPlugin temporarily installs a single test plugin into the
// package-level registry and restores the original state on cleanup.
// This is required because ReadFile, Open, Stat, and GetExportedFunction
// all iterate the package-level `registry` global rather than the receiver.
func withRegistryPlugin(t *testing.T) *Plugin {
	t.Helper()
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})
	registry = pluginRegistry{}
	registrationOpen = false // bypass New(); construct directly
	p := newTestPlugin(t)
	registry = pluginRegistry{p}
	return p
}

// ---------------------------------------------------------------------------
// AttachFileSystem
// ---------------------------------------------------------------------------

func TestAttachFileSystem_PopulatesFilePaths(t *testing.T) {
	p := newTestPlugin(t)

	known := p.files.KnownPaths()
	sort.Strings(known)

	// datafiles/ prefix is stripped; data-overlays/ prefix is preserved.
	assert.Contains(t, known, "html/public/index.html")
	assert.Contains(t, known, "items/test-sword.yaml")
	assert.Contains(t, known, "data-overlays/config.yaml")
}

func TestAttachFileSystem_DoesNotRegisterDirectories(t *testing.T) {
	p := newTestPlugin(t)

	for _, k := range p.files.KnownPaths() {
		assert.False(t, strings.HasSuffix(k, "/"), "directory registered as file path: %q", k)
	}
}

// ---------------------------------------------------------------------------
// PluginFiles.ReadFile
// ---------------------------------------------------------------------------

func TestPluginFiles_ReadFile_Found(t *testing.T) {
	p := newTestPlugin(t)

	b, err := p.files.ReadFile("html/public/index.html")
	require.NoError(t, err)
	assert.Contains(t, string(b), "hello")
}

func TestPluginFiles_ReadFile_DataOverlay(t *testing.T) {
	p := newTestPlugin(t)

	b, err := p.files.ReadFile("data-overlays/config.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(b), "somekey")
}

func TestPluginFiles_ReadFile_NotFound(t *testing.T) {
	p := newTestPlugin(t)

	_, err := p.files.ReadFile("nonexistent/file.txt")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

// ReadFile must normalise backslash paths (Windows-style) to forward slashes
// so that the lookup against the embed.FS key succeeds on all platforms.
func TestPluginFiles_ReadFile_BackslashPath(t *testing.T) {
	p := newTestPlugin(t)

	// filepath.Join uses the OS separator, which is backslash on Windows.
	nativePath := filepath.Join("html", "public", "index.html")

	b, err := p.files.ReadFile(nativePath)
	require.NoError(t, err)
	assert.Contains(t, string(b), "hello")
}

// Double-slash paths must also be tolerated.
func TestPluginFiles_ReadFile_DoubleSlash(t *testing.T) {
	p := newTestPlugin(t)

	b, err := p.files.ReadFile("html//public//index.html")
	require.NoError(t, err)
	assert.Contains(t, string(b), "hello")
}

// ---------------------------------------------------------------------------
// PluginFiles.Open
// ---------------------------------------------------------------------------

func TestPluginFiles_Open_Found(t *testing.T) {
	p := newTestPlugin(t)

	f, err := p.files.Open("items/test-sword.yaml")
	require.NoError(t, err)
	defer f.Close()

	info, err := f.Stat()
	require.NoError(t, err)
	assert.Equal(t, "test-sword.yaml", info.Name())
	assert.False(t, info.IsDir())
}

func TestPluginFiles_Open_NotFound(t *testing.T) {
	p := newTestPlugin(t)

	_, err := p.files.Open("no/such/file.txt")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestPluginFiles_Open_BackslashPath(t *testing.T) {
	p := newTestPlugin(t)

	nativePath := filepath.Join("items", "test-sword.yaml")
	f, err := p.files.Open(nativePath)
	require.NoError(t, err)
	f.Close()
}

// ---------------------------------------------------------------------------
// PluginFiles.Stat
// ---------------------------------------------------------------------------

func TestPluginFiles_Stat_Found(t *testing.T) {
	p := newTestPlugin(t)

	info, err := p.files.Stat("items/test-sword.yaml")
	require.NoError(t, err)
	assert.Equal(t, "test-sword.yaml", info.Name())
	assert.Greater(t, info.Size(), int64(0))
}

func TestPluginFiles_Stat_NotFound(t *testing.T) {
	p := newTestPlugin(t)

	_, err := p.files.Stat("ghost/file.txt")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestPluginFiles_Stat_BackslashPath(t *testing.T) {
	p := newTestPlugin(t)

	nativePath := filepath.Join("items", "test-sword.yaml")
	info, err := p.files.Stat(nativePath)
	require.NoError(t, err)
	assert.Equal(t, "test-sword.yaml", info.Name())
}

// ---------------------------------------------------------------------------
// pluginRegistry.ReadFile / Open / Stat
// NOTE: these methods range over the package-level `registry` global, not the
// receiver, so tests must install the plugin into that global.
// ---------------------------------------------------------------------------

func TestRegistry_ReadFile_Found(t *testing.T) {
	withRegistryPlugin(t)

	b, err := registry.ReadFile("html/public/index.html")
	require.NoError(t, err)
	assert.Contains(t, string(b), "hello")
}

func TestRegistry_ReadFile_NotFound(t *testing.T) {
	withRegistryPlugin(t)

	_, err := registry.ReadFile("missing.txt")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestRegistry_Open_Found(t *testing.T) {
	withRegistryPlugin(t)

	f, err := registry.Open("items/test-sword.yaml")
	require.NoError(t, err)
	f.Close()
}

func TestRegistry_Open_NotFound(t *testing.T) {
	withRegistryPlugin(t)

	_, err := registry.Open("missing.txt")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestRegistry_Stat_Found(t *testing.T) {
	withRegistryPlugin(t)

	info, err := registry.Stat("items/test-sword.yaml")
	require.NoError(t, err)
	assert.Equal(t, "test-sword.yaml", info.Name())
}

func TestRegistry_Stat_NotFound(t *testing.T) {
	withRegistryPlugin(t)

	_, err := registry.Stat("missing.txt")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

// ---------------------------------------------------------------------------
// pluginRegistry.AllFileSubSystems
// AllFileSubSystems uses the receiver, so isolated registries work fine here.
// ---------------------------------------------------------------------------

func TestRegistry_AllFileSubSystems_IteratesAll(t *testing.T) {
	p1 := newTestPlugin(t)
	p2 := newTestPlugin(t)
	reg := pluginRegistry{p1, p2}

	var visited int
	reg.AllFileSubSystems(func(f fs.ReadFileFS) bool {
		visited++
		return true
	})
	assert.Equal(t, 2, visited)
}

func TestRegistry_AllFileSubSystems_EarlyStop(t *testing.T) {
	p1 := newTestPlugin(t)
	p2 := newTestPlugin(t)
	p3 := newTestPlugin(t)
	reg := pluginRegistry{p1, p2, p3}

	var visited int
	reg.AllFileSubSystems(func(f fs.ReadFileFS) bool {
		visited++
		return visited < 2 // stop after second
	})
	assert.Equal(t, 2, visited)
}

// ---------------------------------------------------------------------------
// pluginRegistry first-wins semantics
// ---------------------------------------------------------------------------

// When two plugins both expose the same short path, the first registry entry
// wins (ReadFile returns the first match and does not error).
func TestRegistry_ReadFile_FirstPluginWins(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})
	p1 := newTestPlugin(t)
	p2 := newTestPlugin(t)
	registry = pluginRegistry{p1, p2}

	b, err := registry.ReadFile("html/public/index.html")
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

// ---------------------------------------------------------------------------
// WriteBytes / ReadBytes round-trip
// ---------------------------------------------------------------------------

func TestPlugin_WriteBytesReadBytes_RoundTrip(t *testing.T) {
	p := &Plugin{name: "roundtrip", version: "0.1"}
	p.files.filePaths = map[string]string{}

	origWriteFolder := writeFolderPath
	writeFolderPath = t.TempDir()
	t.Cleanup(func() { writeFolderPath = origWriteFolder })

	payload := []byte("hello plugin data")
	require.NoError(t, p.WriteBytes("mydata", payload))

	got, err := p.ReadBytes("mydata")
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestPlugin_ReadBytes_NotExist(t *testing.T) {
	p := &Plugin{name: "readmissing", version: "0.1"}

	origWriteFolder := writeFolderPath
	writeFolderPath = t.TempDir()
	t.Cleanup(func() { writeFolderPath = origWriteFolder })

	_, err := p.ReadBytes("nope")
	assert.True(t, errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err))
}

// Identifiers with special characters must be sanitised into a valid file name
// and still round-trip correctly.
func TestPlugin_WriteBytes_IdentifierSanitised(t *testing.T) {
	p := &Plugin{name: "sanitise", version: "1.0"}

	origWriteFolder := writeFolderPath
	writeFolderPath = t.TempDir()
	t.Cleanup(func() { writeFolderPath = origWriteFolder })

	require.NoError(t, p.WriteBytes("my data/key:value", []byte("ok")))

	got, err := p.ReadBytes("my data/key:value")
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), got)
}

// ---------------------------------------------------------------------------
// WriteStruct / ReadIntoStruct round-trip
// ---------------------------------------------------------------------------

type testStructPayload struct {
	Name  string `yaml:"name"`
	Value int    `yaml:"value"`
}

func TestPlugin_WriteStructReadIntoStruct_RoundTrip(t *testing.T) {
	p := &Plugin{name: "structtest", version: "2.0"}

	origWriteFolder := writeFolderPath
	writeFolderPath = t.TempDir()
	t.Cleanup(func() { writeFolderPath = origWriteFolder })

	in := testStructPayload{Name: "sword", Value: 42}
	require.NoError(t, p.WriteStruct("weapon", in))

	var out testStructPayload
	require.NoError(t, p.ReadIntoStruct("weapon", &out))
	assert.Equal(t, in, out)
}

// ---------------------------------------------------------------------------
// New / name sanitisation
// ---------------------------------------------------------------------------

func TestNew_NameSanitisation(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})

	registry = pluginRegistry{}
	registrationOpen = true

	p := New("my-plugin name!", "1.0")
	require.NotNil(t, p)
	assert.Equal(t, "my_plugin_name_", p.name)
}

func TestNew_RegistrationClosed(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})

	registrationOpen = false
	p := New("shouldfail", "1.0")
	assert.Nil(t, p)
}

// ---------------------------------------------------------------------------
// ReserveTags / GetRegisteredRoomTags
// ---------------------------------------------------------------------------

func TestReserveTags_GetRegisteredRoomTags(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})

	registry = pluginRegistry{}
	registrationOpen = true

	p := New("tagtest", "1.0")
	require.NotNil(t, p)
	p.ReserveTags("shop", "quest-giver")

	tags := GetRegisteredRoomTags()
	require.Contains(t, tags, "tagtest")
	assert.ElementsMatch(t, []string{"shop", "quest-giver"}, tags["tagtest"])
}

func TestGetRegisteredRoomTags_EmptyWhenNoTags(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})

	registry = pluginRegistry{}
	registrationOpen = true

	New("notags", "1.0")

	tags := GetRegisteredRoomTags()
	assert.NotContains(t, tags, "notags")
}

// ---------------------------------------------------------------------------
// ExportFunction
// ---------------------------------------------------------------------------

func TestExportFunction_PanicsOnNonFunction(t *testing.T) {
	p := &Plugin{name: "exptest", version: "1.0"}
	assert.Panics(t, func() {
		p.ExportFunction("notafunc", "a string")
	})
}

func TestExportFunction_RegistersFunction(t *testing.T) {
	p := &Plugin{name: "exptest2", version: "1.0"}
	fn := func() string { return "ok" }
	p.ExportFunction("myFunc", fn)
	require.NotNil(t, p.exportedFunctions)
	assert.NotNil(t, p.exportedFunctions["myFunc"])
}

// ---------------------------------------------------------------------------
// pluginRegistry.GetExportedFunction
// ---------------------------------------------------------------------------

// GetExportedFunction also ranges over the package-level registry global.
func TestRegistry_GetExportedFunction_Found(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})
	p := &Plugin{name: "expfind", version: "1.0"}
	fn := func() int { return 7 }
	p.ExportFunction("compute", fn)
	registry = pluginRegistry{p}

	got, ok := registry.GetExportedFunction("compute")
	assert.True(t, ok)
	assert.NotNil(t, got)
}

func TestRegistry_GetExportedFunction_NotFound(t *testing.T) {
	origRegistry := registry
	origOpen := registrationOpen
	t.Cleanup(func() {
		registry = origRegistry
		registrationOpen = origOpen
	})
	registry = pluginRegistry{&Plugin{name: "empty", version: "1.0"}}
	_, ok := registry.GetExportedFunction("ghost")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Requires dependency tracking
// ---------------------------------------------------------------------------

func TestRequires_NameSanitisation(t *testing.T) {
	p := &Plugin{name: "deptest", version: "1.0"}
	p.Requires("some-dep name!", "2.0")
	require.Len(t, p.dependencies, 1)
	assert.Equal(t, "some_dep_name_", p.dependencies[0].name)
	assert.Equal(t, "2.0", p.dependencies[0].version)
}
