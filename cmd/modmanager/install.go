package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// cmdInstall installs a module by name from the registry.
func cmdInstall(name string) error {
	if err := validateName(name); err != nil {
		return err
	}

	reg, err := fetchRegistry()
	if err != nil {
		return err
	}

	entry, err := reg.findEntry(name)
	if err != nil {
		return err
	}

	lf, err := readLockFile()
	if err != nil {
		return err
	}

	if existing := lf.findLocked(name); existing != nil && existing.Version == entry.Version {
		fmt.Printf("%s Module %s is already installed at version %s.\n", green("✓"), bold(name), cyan(entry.Version))
		return nil
	}

	printStep("Downloading %s %s...", bold(entry.Name), cyan("v"+entry.Version))

	tmpFile, err := os.CreateTemp("", "gomud-module-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	resp, err := http.Get(entry.URL)
	if err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading archive: HTTP %d from %s", resp.StatusCode, entry.URL)
	}

	if err := verifyArchive(resp.Body, tmpFile, entry.SHA256); err != nil {
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("flushing temp file: %w", err)
	}

	destDir := filepath.Join("modules", name)
	if _, err := os.Stat(destDir); err == nil {
		fmt.Printf("%s Removing existing %s...\n", yellow("-"), gray(destDir))
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("removing existing module directory: %w", err)
		}
	}

	printStep("Extracting to %s...", gray(destDir))

	archiveType, err := detectArchiveType(entry.URL, tmpPath)
	if err != nil {
		return err
	}

	switch archiveType {
	case "targz":
		err = extractTarGz(tmpPath, destDir)
	case "zip":
		err = extractZip(tmpPath, destDir)
	default:
		return fmt.Errorf("unsupported archive format")
	}
	if err != nil {
		return err
	}

	lf.upsert(newLockEntry(entry))
	if err := writeLockFile(lf); err != nil {
		return err
	}

	printInstallNextSteps(name)
	return nil
}

// detectArchiveType returns "targz" or "zip" based on URL suffix first, then
// by sniffing the first bytes of the file.
func detectArchiveType(url, path string) (string, error) {
	lower := strings.ToLower(url)
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return "targz", nil
	}
	if strings.HasSuffix(lower, ".zip") {
		return "zip", nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening archive for type detection: %w", err)
	}
	defer f.Close()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return "", fmt.Errorf("reading archive header: %w", err)
	}

	// gzip magic: 0x1f 0x8b
	if magic[0] == 0x1f && magic[1] == 0x8b {
		return "targz", nil
	}
	// zip magic: PK (0x50 0x4b)
	if magic[0] == 0x50 && magic[1] == 0x4b {
		return "zip", nil
	}

	return "", fmt.Errorf("cannot detect archive type: unknown magic bytes %x", magic[:2])
}

// extractTarGz extracts a .tar.gz archive into destDir, stripping one leading
// path component if all entries share a common top-level directory.
func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// Collect all entries first to determine the common prefix.
	type tarEntry struct {
		header *tar.Header
		data   []byte
	}
	var entries []tarEntry

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		var data []byte
		if hdr.Typeflag == tar.TypeReg {
			data, err = io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("reading tar entry %q: %w", hdr.Name, err)
			}
		}
		entries = append(entries, tarEntry{hdr, data})
	}

	prefix := commonPrefix(func() []string {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.header.Name
		}
		return names
	}())

	for _, e := range entries {
		rel := strings.TrimPrefix(e.header.Name, prefix)
		if rel == "" || rel == "." {
			continue
		}
		if err := writeExtractedEntry(destDir, rel, e.header.FileInfo().IsDir(), e.data, e.header.FileInfo().Mode()); err != nil {
			return err
		}
	}
	return nil
}

// extractZip extracts a .zip archive into destDir, stripping one leading path
// component if all entries share a common top-level directory.
func extractZip(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer zr.Close()

	prefix := commonPrefix(func() []string {
		names := make([]string, len(zr.File))
		for i, f := range zr.File {
			names[i] = f.Name
		}
		return names
	}())

	for _, zf := range zr.File {
		rel := strings.TrimPrefix(zf.Name, prefix)
		if rel == "" || rel == "." {
			continue
		}

		var data []byte
		if !zf.FileInfo().IsDir() {
			rc, err := zf.Open()
			if err != nil {
				return fmt.Errorf("opening zip entry %q: %w", zf.Name, err)
			}
			data, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("reading zip entry %q: %w", zf.Name, err)
			}
		}

		if err := writeExtractedEntry(destDir, rel, zf.FileInfo().IsDir(), data, zf.FileInfo().Mode()); err != nil {
			return err
		}
	}
	return nil
}

// writeExtractedEntry writes a single file or directory entry from an archive
// into destDir. It guards against path traversal attacks.
func writeExtractedEntry(destDir, rel string, isDir bool, data []byte, mode os.FileMode) error {
	// Sanitise the relative path and guard against traversal.
	rel = filepath.Clean(rel)
	if strings.HasPrefix(rel, "..") {
		return fmt.Errorf("archive entry %q would escape destination directory (path traversal)", rel)
	}

	target := filepath.Join(destDir, rel)

	// Double-check after Join that target is still inside destDir.
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", target, err)
	}
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolving dest dir %q: %w", destDir, err)
	}
	if !strings.HasPrefix(absTarget, absDest+string(filepath.Separator)) && absTarget != absDest {
		return fmt.Errorf("archive entry %q would escape destination directory (path traversal)", rel)
	}

	if isDir {
		return os.MkdirAll(target, 0755)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("creating parent directory for %q: %w", target, err)
	}

	perm := mode & 0777
	if perm == 0 {
		perm = 0644
	}

	if err := os.WriteFile(target, data, perm); err != nil {
		return fmt.Errorf("writing %q: %w", target, err)
	}
	return nil
}

// commonPrefix returns the single leading path component shared by all entries,
// including the trailing slash, so it can be stripped with TrimPrefix. Returns
// an empty string if there is no common prefix or only one component.
func commonPrefix(names []string) string {
	if len(names) == 0 {
		return ""
	}

	// Find the first path component of the first entry.
	first := filepath.ToSlash(names[0])
	idx := strings.Index(first, "/")
	if idx < 0 {
		return ""
	}
	candidate := first[:idx+1] // e.g. "birds-1.0.0/"

	for _, name := range names[1:] {
		if !strings.HasPrefix(filepath.ToSlash(name), candidate) {
			return ""
		}
	}
	return candidate
}

func printInstallNextSteps(name string) {
	fmt.Println()
	printSuccess("Module %s installed to %s", bold(name), gray("modules/"+name+"/"))
	fmt.Println()
	fmt.Println(bold("To activate, rebuild the server:"))
	fmt.Println(codeSnippet("make build"))
	fmt.Println(codeSnippet("(or: go generate && go build -o go-mud-server)"))
	fmt.Println()
	fmt.Println(gray("If the module requires new Go dependencies, run first:"))
	fmt.Println(codeSnippet("go mod tidy"))
}
