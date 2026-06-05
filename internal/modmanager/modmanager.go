// Package modmanager implements the GoMud community module manager.
// It is invoked via the main binary with: go-mud-server module [subcommand] [args]
package modmanager

import (
	"fmt"
	"os"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Run is the entry point for the module manager subcommand. args should be
// os.Args[2:] (everything after "module").
func Run(args []string) {
	if len(args) == 0 {
		if isInteractiveTerminal() {
			runInteractive()
			return
		}
		printUsage()
		os.Exit(0)
	}

	var err error
	switch args[0] {
	case "list":
		err = cmdList()

	case "info":
		if len(args) < 2 {
			fatalf("usage: module info <name>\n")
		}
		err = cmdInfo(args[1])

	case "install":
		if len(args) < 2 {
			fatalf("usage: module install <name|all-official>\n")
		}
		err = cmdInstall(args[1])

	case "remove":
		if len(args) < 2 {
			fatalf("usage: module remove <name>\n")
		}
		err = cmdRemove(args[1])

	case "update":
		name := ""
		if len(args) >= 2 {
			name = args[1]
		}
		err = cmdUpdate(name)

	case "package":
		if len(args) < 2 {
			fatalf("usage: module package <name>\n")
		}
		err = cmdPackage(args[1])

	default:
		printError("unknown subcommand: %q", args[0])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		printError("%v", err)
		os.Exit(1)
	}
}

// validateName returns an error if name is not a safe module directory name.
func validateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid module name %q: must match [a-z][a-z0-9-]*", name)
	}
	return nil
}

func fatalf(format string, args ...any) {
	printError(format, args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Println()
	fmt.Println(cyan("GoMud Module Manager"))
	fmt.Println()
	fmt.Println(bold("Usage:"))
	fmt.Println("  go-mud-server module " + yellow("<subcommand>") + " [arguments]")
	fmt.Println()
	fmt.Println(bold("Subcommands:"))
	type entry struct{ cmd, desc string }
	entries := []entry{
		{green("list"), "List available modules from the registry"},
		{green("info") + " <name>", "Show details for a module"},
		{green("install") + " <name>", "Download, verify, and install a module"},
		{green("install") + " all-official", "Install all official GoMud modules at once"},
		{green("remove") + " <name>", "Remove an installed module"},
		{green("update") + " [name]", "Check for updates; update a specific module if name given"},
		{green("package") + " <name>", "Package a local module into a .tar.gz and print its SHA256"},
	}
	for _, e := range entries {
		fmt.Printf("  %s  %s\n", padRight(e.cmd, 22), e.desc)
	}
	fmt.Println()
	fmt.Println(gray("Run without arguments (with a terminal) to start interactive mode."))
	fmt.Println()
	fmt.Println(bold("After installing or removing a module, rebuild the server:"))
	fmt.Println(codeSnippet("make build"))
	fmt.Println(codeSnippet("(or: go generate && go build -o go-mud-server)"))
	fmt.Println()
	fmt.Println(gray("Registry: https://raw.githubusercontent.com/GoMudEngine/GoMud-Modules/refs/heads/master/module-registry.yaml"))
	fmt.Println()
}
