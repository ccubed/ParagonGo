package modmanager

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
			"(only modules installed via 'module install' can be removed this way)", name)
	}

	destDir := filepath.Join("modules", name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		printWarning("directory %s does not exist; updating lock file only", destDir)
	} else {
		printStep("Removing %s...", gray(destDir))
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("removing module directory: %w", err)
		}
	}

	lf.remove(name)
	if err := writeLockFile(lf); err != nil {
		return err
	}

	printSuccess("Module %s removed.", bold(name))
	fmt.Println()
	fmt.Println(bold("To deactivate, rebuild the server:"))
	fmt.Println(codeSnippet("make build"))
	fmt.Println(codeSnippet("(or: go generate && go build -o go-mud-server)"))
	return nil
}
