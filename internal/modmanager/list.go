package modmanager

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
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
		printWarning("could not fetch registry: %v", regErr)
		printWarning("showing installed modules only")
		fmt.Println()
		printInstalledOnly(lf)
		return nil
	}

	if lfErr != nil {
		printWarning("could not read lock file: %v", lfErr)
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

	fmt.Println()
	fmt.Printf("  %s  %s\n", padRight(gray("Name:"), 20), bold(entry.Name))
	fmt.Printf("  %s  %s\n", padRight(gray("Version:"), 20), cyan(entry.Version))
	fmt.Printf("  %s  %s\n", padRight(gray("Author:"), 20), entry.Author)
	fmt.Printf("  %s  %s\n", padRight(gray("Description:"), 20), entry.Description)
	fmt.Printf("  %s  %s\n", padRight(gray("URL:"), 20), blue(entry.URL))
	fmt.Printf("  %s  %s\n", padRight(gray("SHA256:"), 20), dimStr(entry.SHA256))
	if installed {
		locked := lf.findLocked(name)
		fmt.Printf("  %s  %s\n", padRight(gray("Installed:"), 20), green(fmt.Sprintf("yes (v%s, %s)", locked.Version, locked.InstalledAt)))
	} else {
		fmt.Printf("  %s  %s\n", padRight(gray("Installed:"), 20), dimStr("no"))
	}
	fmt.Println()
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
		fmt.Println(dimStr("No community modules are installed."))
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
			fmt.Printf("%s %s is up to date (%s).\n", green("✓"), bold(name), cyan("v"+locked.Version))
			return nil
		}
		fmt.Printf("%s Updating %s from %s to %s...\n", yellow("↑"), bold(name), dimStr("v"+locked.Version), cyan("v"+entry.Version))
		return cmdInstall(name)
	}

	// No specific name: report all with available updates.
	anyUpdates := false
	for _, locked := range lf.Installed {
		entry, err := reg.findEntry(locked.Name)
		if err != nil {
			printWarning("%v (skipping)", err)
			continue
		}
		if locked.Version != entry.Version {
			fmt.Printf("  %s  installed: %s  available: %s\n",
				padRight(bold(locked.Name), 22),
				dimStr("v"+locked.Version),
				yellow("v"+entry.Version))
			anyUpdates = true
		}
	}
	if !anyUpdates {
		fmt.Println(green("✓ ") + "All installed modules are up to date.")
	} else {
		fmt.Println()
		fmt.Println(bold("To update a module, run:"))
		fmt.Println(codeSnippet("go-mud-server module install <name>"))
	}
	return nil
}

// wrapDescriptions controls whether long descriptions in the list table are
// wrapped to fit the terminal width. When false, each row prints on a single
// line regardless of description length.
const wrapDescriptions = true

// fallbackWidth is used when the terminal width cannot be determined.
const fallbackWidth = 80

func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return fallbackWidth
	}
	return w
}

func printRegistryTable(reg *Registry, lf *LockFile) {
	// Compute minimum column widths from actual data.
	colName := len("NAME")
	colVersion := len("VERSION")
	colAuthor := len("AUTHOR")
	colStatus := len("STATUS")

	type row struct {
		name, version, author, status, description string
		official                                   bool
	}
	rows := make([]row, 0, len(reg.Modules))
	for _, e := range reg.Modules {
		status := "available"
		if locked := lf.findLocked(e.Name); locked != nil {
			if locked.Version == e.Version {
				status = "installed"
			} else {
				status = "update avail"
			}
		}
		official := e.Author == officialAuthor
		r := row{e.Name, e.Version, e.Author, status, e.Description, official}
		rows = append(rows, r)
		if len(r.name) > colName {
			colName = len(r.name)
		}
		if len(r.version) > colVersion {
			colVersion = len(r.version)
		}
		if len(r.author) > colAuthor {
			colAuthor = len(r.author)
		}
		if len(r.status) > colStatus {
			colStatus = len(r.status)
		}
	}

	// Separator widths are based on plain-text column widths.
	totalSep := colName + 2 + colVersion + 2 + colAuthor + 2 + colStatus + 2 + 11 // rough desc header
	if totalSep > terminalWidth() {
		totalSep = terminalWidth()
	}

	fmt.Println()
	fmt.Printf("  %s  %s  %s  %s  %s\n",
		padRight(bold("NAME"), colName),
		padRight(bold("VERSION"), colVersion),
		padRight(bold("AUTHOR"), colAuthor),
		padRight(bold("STATUS"), colStatus),
		bold("DESCRIPTION"),
	)
	fmt.Println("  " + divider(totalSep))

	descIndent := colName + 2 + colVersion + 2 + colAuthor + 2 + colStatus + 2
	indent := strings.Repeat(" ", descIndent+2) // +2 for leading "  "

	for _, r := range rows {
		coloredStatus := statusColor(r.status)
		var authorStr string
		if r.official {
			authorStr = green(r.author)
		} else {
			authorStr = dimStr(r.author)
		}
		if wrapDescriptions {
			descWidth := terminalWidth() - descIndent - 2
			lines := wrapText(r.description, descWidth)
			fmt.Printf("  %s  %s  %s  %s  %s\n",
				padRight(cyan(r.name), colName),
				padRight(dimStr(r.version), colVersion),
				padRight(authorStr, colAuthor),
				padRight(coloredStatus, colStatus),
				lines[0],
			)
			for _, line := range lines[1:] {
				fmt.Printf("%s%s\n", indent, line)
			}
		} else {
			fmt.Printf("  %s  %s  %s  %s  %s\n",
				padRight(cyan(r.name), colName),
				padRight(dimStr(r.version), colVersion),
				padRight(authorStr, colAuthor),
				padRight(coloredStatus, colStatus),
				r.description,
			)
		}
	}
	fmt.Println()
	fmt.Printf("  %s  %s\n",
		magenta("Tip:"),
		"To install all "+bold("official GoMud")+" modules at once: "+cyan("install all-official"),
	)
	fmt.Println()
}

// wrapText splits text into lines of at most width characters, breaking on
// word boundaries. It always returns at least one element.
func wrapText(text string, width int) []string {
	if width <= 0 || len(text) <= width {
		return []string{text}
	}
	var lines []string
	words := strings.Fields(text)
	var current strings.Builder
	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if current.Len()+1+len(word) > width {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
		} else {
			current.WriteByte(' ')
			current.WriteString(word)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func printInstalledOnly(lf *LockFile) {
	if lf == nil || len(lf.Installed) == 0 {
		fmt.Println(dimStr("No community modules are installed."))
		return
	}
	fmt.Println()
	fmt.Printf("  %s  %s  %s\n", padRight(bold("NAME"), 20), padRight(bold("VERSION"), 10), bold("INSTALLED AT"))
	fmt.Println("  " + divider(60))
	for _, e := range lf.Installed {
		fmt.Printf("  %s  %s  %s\n", padRight(cyan(e.Name), 20), padRight(dimStr(e.Version), 10), gray(e.InstalledAt))
	}
	fmt.Println()
}
