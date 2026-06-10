package modmanager

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

const registryURL = "https://raw.githubusercontent.com/GoMudEngine/GoMud-Modules/refs/heads/master/module-registry.yaml"

// officialAuthor is the author value used for modules published by the GoMud team.
const officialAuthor = "GoMud"

// manifestSource is the location the module manifest (registry) is loaded from.
// It defaults to the official registry URL but can be temporarily overridden,
// either to another URL or to a local filesystem path, via the global
// --manifest flag (see useManifestOverride). This is primarily intended for
// local testing of modules and registries.
var manifestSource = registryURL

// RegistryEntry is a single module entry from the central registry.
type RegistryEntry struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	URL         string `yaml:"url"`
	SHA256      string `yaml:"sha256"`
}

// Registry is the top-level structure of module-registry.yaml.
type Registry struct {
	Modules []RegistryEntry `yaml:"modules"`
}

// fetchRegistry loads and parses the module manifest from manifestSource, which
// may be an http(s) URL or a local filesystem path (optionally prefixed with
// file://).
func fetchRegistry() (*Registry, error) {
	var data []byte
	if isHTTPURL(manifestSource) {
		var err error
		data, err = fetchRegistryHTTP(manifestSource)
		if err != nil {
			return nil, err
		}
	} else {
		path := strings.TrimPrefix(manifestSource, "file://")
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading manifest file %q: %w", path, err)
		}
	}

	return parseRegistry(data)
}

// fetchRegistryHTTP downloads the raw manifest bytes from an http(s) URL.
func fetchRegistryHTTP(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching registry: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading registry response: %w", err)
	}

	return data, nil
}

// isHTTPURL reports whether s is an http(s) URL (as opposed to a local path).
func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// openArchiveReader returns a reader for a module archive at the given location,
// which may be an http(s) URL or a local filesystem path (optionally prefixed
// with file://). The returned ReadCloser must be closed by the caller. Local
// paths let modules be installed from a manifest used for local testing.
func openArchiveReader(location string) (io.ReadCloser, error) {
	if isHTTPURL(location) {
		resp, err := http.Get(location)
		if err != nil {
			return nil, fmt.Errorf("downloading archive: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("downloading archive: HTTP %d from %s", resp.StatusCode, location)
		}
		return resp.Body, nil
	}

	path := strings.TrimPrefix(location, "file://")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening local archive %q: %w", path, err)
	}
	return f, nil
}

// parseRegistry parses raw YAML bytes into a Registry.
func parseRegistry(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry YAML: %w", err)
	}
	return &reg, nil
}

// officialModules returns the subset of registry entries authored by the GoMud team.
func (r *Registry) officialModules() []RegistryEntry {
	var out []RegistryEntry
	for _, e := range r.Modules {
		if e.Author == officialAuthor {
			out = append(out, e)
		}
	}
	return out
}

// findEntry returns the registry entry for the given module name, or an error
// if the name is not found.
func (r *Registry) findEntry(name string) (*RegistryEntry, error) {
	for i := range r.Modules {
		if r.Modules[i].Name == name {
			return &r.Modules[i], nil
		}
	}
	return nil, fmt.Errorf("module %q not found in registry", name)
}

// verifyArchive reads from r, writes to w, and returns an error if the SHA256
// of the bytes read does not match expectedHex. The caller is responsible for
// seeking or re-opening the destination if needed after this call.
func verifyArchive(r io.Reader, w io.Writer, expectedHex string) error {
	expectedHex = strings.ToLower(strings.TrimSpace(expectedHex))

	h := sha256.New()
	mw := io.MultiWriter(w, h)

	if _, err := io.Copy(mw, r); err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expectedHex {
		return fmt.Errorf("SHA256 mismatch:\n  expected: %s\n  actual:   %s", expectedHex, actual)
	}
	return nil
}
