package modmanager

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// cmdPackage creates a .tar.gz archive of an existing module directory and
// prints the filename and its SHA256 digest.
func cmdPackage(name string) error {
	if err := validateName(name); err != nil {
		return err
	}

	srcDir := filepath.Join("modules", name)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("module directory %q does not exist", srcDir)
	}

	outFile := name + ".tar.gz"

	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("creating archive %q: %w", outFile, err)
	}
	defer f.Close()

	h := sha256.New()
	mw := io.MultiWriter(f, h)

	gw := gzip.NewWriter(mw)
	tw := tar.NewWriter(gw)

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute the path inside the archive relative to the module directory.
		// e.g. modules/birds/birds.go -> birds/birds.go
		rel, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return fmt.Errorf("computing relative path for %q: %w", path, err)
		}
		rel = filepath.ToSlash(rel)

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("building tar header for %q: %w", path, err)
		}
		hdr.Name = rel

		if info.IsDir() {
			hdr.Name += "/"
			return tw.WriteHeader(hdr)
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("writing tar header for %q: %w", rel, err)
		}

		src, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %q: %w", path, err)
		}
		defer src.Close()

		if _, err := io.Copy(tw, src); err != nil {
			return fmt.Errorf("writing %q to archive: %w", rel, err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("finalising tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("finalising gzip: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing archive file: %w", err)
	}

	digest := hex.EncodeToString(h.Sum(nil))

	fmt.Println()
	fmt.Printf("  %s  %s\n", padRight(gray("Archive:"), 12), bold(outFile))
	fmt.Printf("  %s  %s\n", padRight(gray("SHA256:"), 12), dimStr(digest))
	fmt.Println()
	return nil
}
