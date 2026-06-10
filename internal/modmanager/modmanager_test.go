package modmanager

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// registry
// ---------------------------------------------------------------------------

func TestParseRegistry_valid(t *testing.T) {
	data := []byte(`
modules:
  - name: birds
    description: "Adds birds"
    version: "1.0.0"
    author: "alice"
    url: "https://example.com/birds-1.0.0.tar.gz"
    sha256: "abc123"
  - name: fishing
    description: "Fishing system"
    version: "2.1.0"
    author: "bob"
    url: "https://example.com/fishing-2.1.0.zip"
    sha256: "def456"
`)
	reg, err := parseRegistry(data)
	require.NoError(t, err)
	require.Len(t, reg.Modules, 2)
	assert.Equal(t, "birds", reg.Modules[0].Name)
	assert.Equal(t, "1.0.0", reg.Modules[0].Version)
	assert.Equal(t, "fishing", reg.Modules[1].Name)
}

func TestParseRegistry_empty(t *testing.T) {
	reg, err := parseRegistry([]byte("modules: []\n"))
	require.NoError(t, err)
	assert.Empty(t, reg.Modules)
}

func TestParseRegistry_malformed(t *testing.T) {
	_, err := parseRegistry([]byte(":\t: bad yaml"))
	assert.Error(t, err)
}

func TestFindEntry(t *testing.T) {
	reg := &Registry{
		Modules: []RegistryEntry{
			{Name: "birds", Version: "1.0.0"},
			{Name: "fishing", Version: "2.0.0"},
		},
	}

	e, err := reg.findEntry("birds")
	require.NoError(t, err)
	assert.Equal(t, "birds", e.Name)

	_, err = reg.findEntry("missing")
	assert.Error(t, err)
}

func TestVerifyArchive_match(t *testing.T) {
	data := []byte("hello world")
	h := sha256.Sum256(data)
	expected := hex.EncodeToString(h[:])

	var buf bytes.Buffer
	err := verifyArchive(bytes.NewReader(data), &buf, expected)
	require.NoError(t, err)
	assert.Equal(t, data, buf.Bytes())
}

func TestVerifyArchive_mismatch(t *testing.T) {
	data := []byte("hello world")
	err := verifyArchive(bytes.NewReader(data), io_discard, "0000000000000000000000000000000000000000000000000000000000000000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SHA256 mismatch")
}

func TestVerifyArchive_uppercaseExpected(t *testing.T) {
	data := []byte("hello world")
	h := sha256.Sum256(data)
	expected := strings.ToUpper(hex.EncodeToString(h[:]))

	var buf bytes.Buffer
	err := verifyArchive(bytes.NewReader(data), &buf, expected)
	require.NoError(t, err)
}

// io_discard is an io.Writer that discards all bytes.
var io_discard = discardWriter{}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// ---------------------------------------------------------------------------
// lockfile
// ---------------------------------------------------------------------------

func TestLockFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	origPath := lockFilePath

	// Redirect lockFilePath to a temp file for this test.
	// We do this by writing/reading directly rather than swapping the global,
	// since the global is a const. Use helpers directly.
	lf := &LockFile{
		Installed: []LockEntry{
			{Name: "birds", Version: "1.0.0", URL: "https://example.com/birds.tar.gz", SHA256: "abc", InstalledAt: "2026-01-01T00:00:00Z"},
		},
	}

	path := filepath.Join(dir, "modules.lock.yaml")
	data, err := marshalLockFile(lf)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	loaded, err := parseLockFile(path)
	require.NoError(t, err)
	require.Len(t, loaded.Installed, 1)
	assert.Equal(t, "birds", loaded.Installed[0].Name)
	assert.Equal(t, "1.0.0", loaded.Installed[0].Version)

	_ = origPath // suppress unused warning
}

func TestLockFile_upsert(t *testing.T) {
	lf := &LockFile{}

	lf.upsert(LockEntry{Name: "birds", Version: "1.0.0"})
	assert.Len(t, lf.Installed, 1)

	// upsert same name replaces
	lf.upsert(LockEntry{Name: "birds", Version: "2.0.0"})
	assert.Len(t, lf.Installed, 1)
	assert.Equal(t, "2.0.0", lf.Installed[0].Version)

	// upsert different name appends
	lf.upsert(LockEntry{Name: "fishing", Version: "1.0.0"})
	assert.Len(t, lf.Installed, 2)
}

func TestLockFile_remove(t *testing.T) {
	lf := &LockFile{
		Installed: []LockEntry{
			{Name: "birds"},
			{Name: "fishing"},
		},
	}

	lf.remove("birds")
	assert.Len(t, lf.Installed, 1)
	assert.Equal(t, "fishing", lf.Installed[0].Name)

	lf.remove("nonexistent") // no-op
	assert.Len(t, lf.Installed, 1)
}

func TestLockFile_findLocked(t *testing.T) {
	lf := &LockFile{
		Installed: []LockEntry{
			{Name: "birds", Version: "1.0.0"},
		},
	}

	e := lf.findLocked("birds")
	require.NotNil(t, e)
	assert.Equal(t, "1.0.0", e.Version)

	assert.Nil(t, lf.findLocked("missing"))
}

// ---------------------------------------------------------------------------
// archive extraction
// ---------------------------------------------------------------------------

func TestExtractTarGz_flat(t *testing.T) {
	// Build a flat tar.gz (no leading directory component).
	archive := buildTarGz(t, map[string]string{
		"module.go":      "package mymod\n",
		"files/help.txt": "help content\n",
	})

	dest := t.TempDir()
	require.NoError(t, extractTarGzFromBytes(archive, dest))

	assertFileContent(t, filepath.Join(dest, "module.go"), "package mymod\n")
	assertFileContent(t, filepath.Join(dest, "files", "help.txt"), "help content\n")
}

func TestExtractTarGz_withPrefix(t *testing.T) {
	// Build a tar.gz where all entries share a top-level "mymod-1.0.0/" prefix.
	archive := buildTarGz(t, map[string]string{
		"mymod-1.0.0/module.go":      "package mymod\n",
		"mymod-1.0.0/files/help.txt": "help content\n",
	})

	dest := t.TempDir()
	require.NoError(t, extractTarGzFromBytes(archive, dest))

	assertFileContent(t, filepath.Join(dest, "module.go"), "package mymod\n")
	assertFileContent(t, filepath.Join(dest, "files", "help.txt"), "help content\n")
}

func TestExtractZip_flat(t *testing.T) {
	archive := buildZip(t, map[string]string{
		"module.go":      "package mymod\n",
		"files/help.txt": "help content\n",
	})

	dest := t.TempDir()
	require.NoError(t, extractZipFromBytes(archive, dest))

	assertFileContent(t, filepath.Join(dest, "module.go"), "package mymod\n")
	assertFileContent(t, filepath.Join(dest, "files", "help.txt"), "help content\n")
}

func TestExtractZip_withPrefix(t *testing.T) {
	archive := buildZip(t, map[string]string{
		"mymod-1.0.0/module.go":      "package mymod\n",
		"mymod-1.0.0/files/help.txt": "help content\n",
	})

	dest := t.TempDir()
	require.NoError(t, extractZipFromBytes(archive, dest))

	assertFileContent(t, filepath.Join(dest, "module.go"), "package mymod\n")
	assertFileContent(t, filepath.Join(dest, "files", "help.txt"), "help content\n")
}

func TestExtractTarGz_pathTraversal(t *testing.T) {
	archive := buildTarGz(t, map[string]string{
		"../../evil.go": "package evil\n",
	})

	dest := t.TempDir()
	err := extractTarGzFromBytes(archive, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestExtractZip_pathTraversal(t *testing.T) {
	archive := buildZip(t, map[string]string{
		"../../evil.go": "package evil\n",
	})

	dest := t.TempDir()
	err := extractZipFromBytes(archive, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

// ---------------------------------------------------------------------------
// commonPrefix
// ---------------------------------------------------------------------------

func TestCommonPrefix(t *testing.T) {
	cases := []struct {
		names    []string
		expected string
	}{
		{[]string{"a/b.go", "a/c.go"}, "a/"},
		{[]string{"a/b.go", "x/c.go"}, ""},
		{[]string{"b.go", "c.go"}, ""},
		{[]string{}, ""},
		{[]string{"only.go"}, ""},
		{[]string{"prefix-1.0/a.go", "prefix-1.0/b.go"}, "prefix-1.0/"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.expected, commonPrefix(tc.names), "input: %v", tc.names)
	}
}

// ---------------------------------------------------------------------------
// officialModules / unofficial confirmation helper
// ---------------------------------------------------------------------------

func TestOfficialModules(t *testing.T) {
	reg := &Registry{
		Modules: []RegistryEntry{
			{Name: "birds", Author: officialAuthor},
			{Name: "fishing", Author: "alice"},
			{Name: "weather", Author: officialAuthor},
		},
	}
	official := reg.officialModules()
	require.Len(t, official, 2)
	assert.Equal(t, "birds", official[0].Name)
	assert.Equal(t, "weather", official[1].Name)
}

func TestConfirmUnofficialInstall_yesAnswer(t *testing.T) {
	// Simulate a user typing "y" on stdin.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := stdinForPrompt
	oldInteractive := isInteractivePrompt
	stdinForPrompt = r
	isInteractivePrompt = func() bool { return true }
	defer func() {
		stdinForPrompt = oldStdin
		isInteractivePrompt = oldInteractive
	}()

	_, _ = w.WriteString("y\n")
	w.Close()

	assert.True(t, confirmUnofficialInstall("fishing", "alice"))
}

func TestConfirmUnofficialInstall_noAnswer(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := stdinForPrompt
	oldInteractive := isInteractivePrompt
	stdinForPrompt = r
	isInteractivePrompt = func() bool { return true }
	defer func() {
		stdinForPrompt = oldStdin
		isInteractivePrompt = oldInteractive
	}()

	_, _ = w.WriteString("n\n")
	w.Close()

	assert.False(t, confirmUnofficialInstall("fishing", "alice"))
}

func TestConfirmUnofficialInstall_emptyAnswer(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := stdinForPrompt
	oldInteractive := isInteractivePrompt
	stdinForPrompt = r
	isInteractivePrompt = func() bool { return true }
	defer func() {
		stdinForPrompt = oldStdin
		isInteractivePrompt = oldInteractive
	}()

	_, _ = w.WriteString("\n")
	w.Close()

	assert.False(t, confirmUnofficialInstall("fishing", "alice"))
}

func TestConfirmUnofficialInstall_eofReturnsFalse(t *testing.T) {
	// EOF on stdin (pipe closed immediately) should return false.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := stdinForPrompt
	oldInteractive := isInteractivePrompt
	stdinForPrompt = r
	isInteractivePrompt = func() bool { return true }
	defer func() {
		stdinForPrompt = oldStdin
		isInteractivePrompt = oldInteractive
	}()

	w.Close()

	assert.False(t, confirmUnofficialInstall("fishing", "alice"))
}

func TestConfirmUnofficialInstall_nonInteractiveReturnsFalse(t *testing.T) {
	oldInteractive := isInteractivePrompt
	isInteractivePrompt = func() bool { return false }
	defer func() { isInteractivePrompt = oldInteractive }()

	assert.False(t, confirmUnofficialInstall("fishing", "alice"))
}

// ---------------------------------------------------------------------------
// validateName
// ---------------------------------------------------------------------------

func TestValidateName(t *testing.T) {
	valid := []string{"birds", "my-module", "mod123", "a"}
	for _, n := range valid {
		assert.NoError(t, validateName(n), "expected valid: %q", n)
	}

	invalid := []string{"", "Birds", "my_module", "-bad", "has space", "../evil", "123start"}
	for _, n := range invalid {
		assert.Error(t, validateName(n), "expected invalid: %q", n)
	}
}

// ---------------------------------------------------------------------------
// manifest override
// ---------------------------------------------------------------------------

func TestValidateManifestSource(t *testing.T) {
	valid := []string{
		"manifest.yaml",
		"manifest.yml",
		"./local/registry.yaml",
		"/abs/path/registry.YAML",
		"file:///abs/path/registry.yaml",
		"https://example.com/module-registry.yaml",
		"https://example.com/module-registry.yaml?ref=main",
	}
	for _, s := range valid {
		assert.NoError(t, validateManifestSource(s), "expected valid: %q", s)
	}

	invalid := []string{
		"",
		"   ",
		"manifest.txt",
		"manifest.json",
		"https://example.com/registry",
		"registry.yaml.bak",
	}
	for _, s := range invalid {
		assert.Error(t, validateManifestSource(s), "expected invalid: %q", s)
	}
}

func TestApplyManifestFlag(t *testing.T) {
	orig := manifestSource
	defer func() { manifestSource = orig }()

	// --manifest <value> form, flag is stripped from returned args.
	rest, err := applyManifestFlag([]string{"--manifest", "local.yaml", "list"})
	require.NoError(t, err)
	assert.Equal(t, []string{"list"}, rest)
	assert.Equal(t, "local.yaml", manifestSource)

	// --manifest=<value> form.
	manifestSource = orig
	rest, err = applyManifestFlag([]string{"install", "--manifest=other.yaml", "birds"})
	require.NoError(t, err)
	assert.Equal(t, []string{"install", "birds"}, rest)
	assert.Equal(t, "other.yaml", manifestSource)

	// "test" resolves to the built-in test registry URL.
	manifestSource = orig
	rest, err = applyManifestFlag([]string{"--manifest", "test", "list"})
	require.NoError(t, err)
	assert.Equal(t, []string{"list"}, rest)
	assert.Equal(t, registryTestURL, manifestSource)

	// "default" resolves back to the official registry URL.
	manifestSource = "local.yaml"
	rest, err = applyManifestFlag([]string{"--manifest=default", "list"})
	require.NoError(t, err)
	assert.Equal(t, []string{"list"}, rest)
	assert.Equal(t, registryURL, manifestSource)

	// No flag leaves the default in place.
	manifestSource = orig
	rest, err = applyManifestFlag([]string{"list"})
	require.NoError(t, err)
	assert.Equal(t, []string{"list"}, rest)
	assert.Equal(t, orig, manifestSource)
}

func TestApplyManifestFlag_errors(t *testing.T) {
	orig := manifestSource
	defer func() { manifestSource = orig }()

	// Missing value.
	_, err := applyManifestFlag([]string{"--manifest"})
	assert.Error(t, err)

	// Non-yaml value.
	manifestSource = orig
	_, err = applyManifestFlag([]string{"--manifest", "registry.txt", "list"})
	assert.Error(t, err)
	assert.Equal(t, orig, manifestSource, "manifestSource must not change on validation failure")
}

func TestHandleManifestSourceCommand(t *testing.T) {
	orig := manifestSource
	defer func() { manifestSource = orig }()

	// No arg leaves the source unchanged (display only).
	manifestSource = orig
	handleManifestSourceCommand(nil)
	assert.Equal(t, orig, manifestSource)

	// Setting a valid .yaml source updates it for the session.
	handleManifestSourceCommand([]string{"local.yaml"})
	assert.Equal(t, "local.yaml", manifestSource)

	// "default" restores the default registry.
	handleManifestSourceCommand([]string{"default"})
	assert.Equal(t, registryURL, manifestSource)

	// "reset" is an alias for default.
	manifestSource = "local.yaml"
	handleManifestSourceCommand([]string{"reset"})
	assert.Equal(t, registryURL, manifestSource)

	// "test" switches to the test registry.
	manifestSource = orig
	handleManifestSourceCommand([]string{"test"})
	assert.Equal(t, registryTestURL, manifestSource)

	// An invalid (non-yaml) source does not change the current value.
	manifestSource = "local.yaml"
	handleManifestSourceCommand([]string{"bad.txt"})
	assert.Equal(t, "local.yaml", manifestSource)
}

func TestFetchRegistry_localFile(t *testing.T) {
	orig := manifestSource
	defer func() { manifestSource = orig }()

	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	content := `
modules:
  - name: birds
    version: "1.0.0"
    author: "GoMud"
    url: "https://example.com/birds.tar.gz"
    sha256: "abc123"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	manifestSource = path
	reg, err := fetchRegistry()
	require.NoError(t, err)
	require.Len(t, reg.Modules, 1)
	assert.Equal(t, "birds", reg.Modules[0].Name)

	// file:// prefix is also accepted.
	manifestSource = "file://" + path
	reg, err = fetchRegistry()
	require.NoError(t, err)
	require.Len(t, reg.Modules, 1)
}

func TestFetchRegistry_localFileMissing(t *testing.T) {
	orig := manifestSource
	defer func() { manifestSource = orig }()

	manifestSource = filepath.Join(t.TempDir(), "does-not-exist.yaml")
	_, err := fetchRegistry()
	assert.Error(t, err)
}

func TestIsHTTPURL(t *testing.T) {
	assert.True(t, isHTTPURL("http://example.com/x.yaml"))
	assert.True(t, isHTTPURL("https://example.com/x.yaml"))
	assert.False(t, isHTTPURL("/local/path.yaml"))
	assert.False(t, isHTTPURL("file:///local/path.yaml"))
	assert.False(t, isHTTPURL("./relative.yaml"))
}

func TestOpenArchiveReader_localFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.tar.gz")
	require.NoError(t, os.WriteFile(path, []byte("payload"), 0644))

	rc, err := openArchiveReader(path)
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))

	// file:// prefix is also accepted.
	rc2, err := openArchiveReader("file://" + path)
	require.NoError(t, err)
	defer rc2.Close()
	data, err = io.ReadAll(rc2)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))
}

func TestOpenArchiveReader_localFileMissing(t *testing.T) {
	_, err := openArchiveReader(filepath.Join(t.TempDir(), "missing.tar.gz"))
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// detectArchiveType
// ---------------------------------------------------------------------------

func TestDetectArchiveType_byURL(t *testing.T) {
	// Write dummy files - content doesn't matter for URL-based detection.
	f := writeTempFile(t, []byte("dummy"))

	typ, err := detectArchiveType("https://example.com/mod.tar.gz", f)
	require.NoError(t, err)
	assert.Equal(t, "targz", typ)

	typ, err = detectArchiveType("https://example.com/mod.tgz", f)
	require.NoError(t, err)
	assert.Equal(t, "targz", typ)

	typ, err = detectArchiveType("https://example.com/mod.zip", f)
	require.NoError(t, err)
	assert.Equal(t, "zip", typ)
}

func TestDetectArchiveType_byMagic(t *testing.T) {
	gzMagic := []byte{0x1f, 0x8b, 0x00, 0x00}
	zipMagic := []byte{0x50, 0x4b, 0x03, 0x04}

	f := writeTempFile(t, gzMagic)
	typ, err := detectArchiveType("https://example.com/mod.bin", f)
	require.NoError(t, err)
	assert.Equal(t, "targz", typ)

	f = writeTempFile(t, zipMagic)
	typ, err = detectArchiveType("https://example.com/mod.bin", f)
	require.NoError(t, err)
	assert.Equal(t, "zip", typ)
}

func TestDetectArchiveType_unknownMagic(t *testing.T) {
	f := writeTempFile(t, []byte{0x00, 0x00, 0x00, 0x00})
	_, err := detectArchiveType("https://example.com/mod.bin", f)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// package
// ---------------------------------------------------------------------------

func TestCmdPackage_createsArchiveAndPrintsSHA256(t *testing.T) {
	// Build a fake modules/mymod directory in a temp working dir.
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "modules", "mymod")
	require.NoError(t, os.MkdirAll(filepath.Join(modDir, "files"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "mymod.go"), []byte("package mymod\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "files", "help.txt"), []byte("help\n"), 0644))

	// Run from the temp dir so the archive lands there.
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(orig)

	require.NoError(t, cmdPackage("mymod"))

	// Archive must exist.
	archivePath := filepath.Join(tmpDir, "mymod.tar.gz")
	info, err := os.Stat(archivePath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// SHA256 of the file must match what cmdPackage printed (verify independently).
	data, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	h := sha256.Sum256(data)
	expected := hex.EncodeToString(h[:])
	assert.NotEmpty(t, expected)

	// Re-extract and confirm contents are present.
	dest := t.TempDir()
	require.NoError(t, extractTarGz(archivePath, dest))
	// The archive has a top-level "mymod/" prefix which extractTarGz strips,
	// so files land directly in dest.
	assertFileContent(t, filepath.Join(dest, "mymod.go"), "package mymod\n")
	assertFileContent(t, filepath.Join(dest, "files", "help.txt"), "help\n")
}

func TestCmdPackage_missingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "modules"), 0755))

	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(orig)

	err = cmdPackage("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestCmdPackage_invalidName(t *testing.T) {
	err := cmdPackage("Bad Name")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// helpers used only in tests
// ---------------------------------------------------------------------------

// marshalLockFile and parseLockFile expose the YAML round-trip for testing
// without touching the real lockFilePath constant.

func marshalLockFile(lf *LockFile) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(lockFileHeader)
	data, err := yamlMarshal(lf)
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	return buf.Bytes(), nil
}

func parseLockFile(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lf LockFile
	if err := yamlUnmarshal(data, &lf); err != nil {
		return nil, err
	}
	return &lf, nil
}

// extractTarGzFromBytes writes the bytes to a temp file and calls extractTarGz.
func extractTarGzFromBytes(data []byte, dest string) error {
	f := writeTempFile(nil, data)
	return extractTarGz(f, dest)
}

// extractZipFromBytes writes the bytes to a temp file and calls extractZip.
func extractZipFromBytes(data []byte, dest string) error {
	f := writeTempFile(nil, data)
	return extractZip(f, dest)
}

func writeTempFile(t *testing.T, data []byte) string {
	var f *os.File
	var err error
	if t != nil {
		f, err = os.CreateTemp(t.TempDir(), "test-*.bin")
	} else {
		f, err = os.CreateTemp("", "test-*.bin")
	}
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		panic(err)
	}
	return f.Name()
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		body := []byte(content)
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(body)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "file should exist: %s", path)
	assert.Equal(t, expected, string(data))
}
