package main

import (
	"fmt"
	"os"
	"strings"
)

// cmdList fetches the registry and prints a table of available modules,
// annotating which ones are currently installed.
func cmdList() error {
	reg, regErr := fetchRegistry()
	lf, lfErr := readLockFile()

	if regErr != nil && lfErr != nil {
		return fmt.Errorf("could not fetch registry (%v) and could not read lock file (%v)", regErr, lfErr)
	}

	if regErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch registry: %v\n", regErr)
		fmt.Fprintf(os.Stderr, "showing installed modules only\n\n")
		printInstalledOnly(lf)
		return nil
	}

	if lfErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read lock file: %v\n", lfErr)
		lf = &LockFile{}
	}

	printRegistryTable(reg, lf)
	return nil
}

// cmdInfo prints full metadata for a single registry entry.
func cmdInfo(name string) error {
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

	lf, _ := readLockFile()
	installed := lf != nil && lf.findLocked(name) != nil

	fmt.Printf("Name:        %s\n", entry.Name)
	fmt.Printf("Version:     %s\n", entry.Version)
	fmt.Printf("Author:      %s\n", entry.Author)
	fmt.Printf("Description: %s\n", entry.Description)
	fmt.Printf("URL:         %s\n", entry.URL)
	fmt.Printf("SHA256:      %s\n", entry.SHA256)
	if installed {
		locked := lf.findLocked(name)
		fmt.Printf("Installed:   yes (v%s, %s)\n", locked.Version, locked.InstalledAt)
	} else {
		fmt.Printf("Installed:   no\n")
	}
	return nil
}

// cmdUpdate checks for updates to installed modules.
// If name is non-empty, only that module is checked and updated.
// If name is empty, all installed modules are checked and any with available
// updates are reported; the operator must run install to apply them.
func cmdUpdate(name string) error {
	reg, err := fetchRegistry()
	if err != nil {
		return err
	}

	lf, err := readLockFile()
	if err != nil {
		return err
	}

	if len(lf.Installed) == 0 {
		fmt.Println("No community modules are installed.")
		return nil
	}

	if name != "" {
		if err := validateName(name); err != nil {
			return err
		}
		locked := lf.findLocked(name)
		if locked == nil {
			return fmt.Errorf("module %q is not installed", name)
		}
		entry, err := reg.findEntry(name)
		if err != nil {
			return err
		}
		if locked.Version == entry.Version {
			fmt.Printf("Module %q is up to date (v%s).\n", name, locked.Version)
			return nil
		}
		fmt.Printf("Updating %q from v%s to v%s...\n", name, locked.Version, entry.Version)
		return cmdInstall(name)
	}

	// No specific name: report all with available updates.
	anyUpdates := false
	for _, locked := range lf.Installed {
		entry, err := reg.findEntry(locked.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v (skipping)\n", err)
			continue
		}
		if locked.Version != entry.Version {
			fmt.Printf("  %-20s  installed: v%-10s  available: v%s\n",
				locked.Name, locked.Version, entry.Version)
			anyUpdates = true
		}
	}
	if !anyUpdates {
		fmt.Println("All installed modules are up to date.")
	} else {
		fmt.Println()
		fmt.Println("To update a module, run:")
		fmt.Println("  modmanager install <name>")
	}
	return nil
}

func printRegistryTable(reg *Registry, lf *LockFile) {
	const (
		colName    = 20
		colVersion = 10
		colStatus  = 12
	)

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
		colName, "NAME",
		colVersion, "VERSION",
		colStatus, "STATUS",
		"DESCRIPTION",
	)
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)+20))

	for _, e := range reg.Modules {
		status := "available"
		if locked := lf.findLocked(e.Name); locked != nil {
			if locked.Version == e.Version {
				status = "installed"
			} else {
				status = "update avail"
			}
		}
		desc := e.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Printf("%-*s  %-*s  %-*s  %s\n",
			colName, e.Name,
			colVersion, e.Version,
			colStatus, status,
			desc,
		)
	}
}

func printInstalledOnly(lf *LockFile) {
	if lf == nil || len(lf.Installed) == 0 {
		fmt.Println("No community modules are installed.")
		return
	}
	fmt.Printf("%-20s  %-10s  %s\n", "NAME", "VERSION", "INSTALLED AT")
	fmt.Println(strings.Repeat("-", 60))
	for _, e := range lf.Installed {
		fmt.Printf("%-20s  %-10s  %s\n", e.Name, e.Version, e.InstalledAt)
	}
}
