package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v2"
)

const registryURL = "https://raw.githubusercontent.com/GoMudEngine/GoMud-Modules/refs/heads/master/module-registry.yaml"

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

// fetchRegistry downloads and parses the module registry from the hardcoded URL.
func fetchRegistry() (*Registry, error) {
	resp, err := http.Get(registryURL)
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

	return parseRegistry(data)
}

// parseRegistry parses raw YAML bytes into a Registry.
func parseRegistry(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry YAML: %w", err)
	}
	return &reg, nil
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
