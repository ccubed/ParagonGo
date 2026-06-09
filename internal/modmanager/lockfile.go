package modmanager

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// yamlMarshal and yamlUnmarshal are thin wrappers used by tests to avoid
// importing yaml directly in the test package.
func yamlMarshal(v interface{}) ([]byte, error)      { return yaml.Marshal(v) }
func yamlUnmarshal(data []byte, v interface{}) error { return yaml.Unmarshal(data, v) }

const lockFilePath = "modules/modules.lock.yaml"

const lockFileHeader = "# Managed by: `make module` or `./go-mud-server module`\n# Do not edit manually.\n"

// LockEntry records a single installed community module.
type LockEntry struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	URL         string `yaml:"url"`
	SHA256      string `yaml:"sha256"`
	InstalledAt string `yaml:"installed_at"`
}

// LockFile is the top-level structure of modules/modules.lock.yaml.
type LockFile struct {
	Installed []LockEntry `yaml:"installed"`
}

// readLockFile reads and parses the lock file. Returns an empty LockFile (not
// an error) if the file does not exist yet.
func readLockFile() (*LockFile, error) {
	data, err := os.ReadFile(lockFilePath)
	if os.IsNotExist(err) {
		return &LockFile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	var lf LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}
	return &lf, nil
}

// writeLockFile serialises lf and writes it to the lock file path.
func writeLockFile(lf *LockFile) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("serialising lock file: %w", err)
	}

	content := []byte(lockFileHeader)
	content = append(content, data...)

	if err := os.WriteFile(lockFilePath, content, 0644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}
	return nil
}

// findLocked returns the lock entry for name, or nil if not installed.
func (lf *LockFile) findLocked(name string) *LockEntry {
	for i := range lf.Installed {
		if lf.Installed[i].Name == name {
			return &lf.Installed[i]
		}
	}
	return nil
}

// upsert adds or replaces the lock entry for entry.Name.
func (lf *LockFile) upsert(entry LockEntry) {
	for i := range lf.Installed {
		if lf.Installed[i].Name == entry.Name {
			lf.Installed[i] = entry
			return
		}
	}
	lf.Installed = append(lf.Installed, entry)
}

// remove deletes the lock entry for name. No-op if not present.
func (lf *LockFile) remove(name string) {
	filtered := lf.Installed[:0]
	for _, e := range lf.Installed {
		if e.Name != name {
			filtered = append(filtered, e)
		}
	}
	lf.Installed = filtered
}

// newLockEntry builds a LockEntry from a registry entry, timestamped now.
func newLockEntry(e *RegistryEntry) LockEntry {
	return LockEntry{
		Name:        e.Name,
		Version:     e.Version,
		URL:         e.URL,
		SHA256:      e.SHA256,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}
}
