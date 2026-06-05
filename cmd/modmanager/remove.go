package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// cmdRemove removes an installed community module by name.
func cmdRemove(name string) error {
	if err := validateName(name); err != nil {
		return err
	}

	lf, err := readLockFile()
	if err != nil {
		return err
	}

	if lf.findLocked(name) == nil {
		return fmt.Errorf("module %q is not managed by the module manager\n"+
			"(only modules installed via 'modmanager install' can be removed this way)", name)
	}

	destDir := filepath.Join("modules", name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: directory %s does not exist; updating lock file only\n", destDir)
	} else {
		fmt.Printf("Removing %s...\n", destDir)
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("removing module directory: %w", err)
		}
	}

	lf.remove(name)
	if err := writeLockFile(lf); err != nil {
		return err
	}

	fmt.Printf("\nModule %q removed.\n", name)
	fmt.Println()
	fmt.Println("To deactivate, rebuild the server:")
	fmt.Println("  make build")
	fmt.Println("  (or: go generate && go build -o go-mud-server)")
	return nil
}
