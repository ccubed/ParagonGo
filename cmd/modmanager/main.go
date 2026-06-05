package main

import (
	"fmt"
	"os"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func main() {
	if len(os.Args) < 2 {
		if isInteractiveTerminal() {
			runInteractive()
			return
		}
		printUsage()
		os.Exit(0)
	}

	var err error
	switch os.Args[1] {
	case "list":
		err = cmdList()

	case "info":
		if len(os.Args) < 3 {
			fatalf("usage: modmanager info <name>\n")
		}
		err = cmdInfo(os.Args[2])

	case "install":
		if len(os.Args) < 3 {
			fatalf("usage: modmanager install <name|all-official>\n")
		}
		err = cmdInstall(os.Args[2])

	case "remove":
		if len(os.Args) < 3 {
			fatalf("usage: modmanager remove <name>\n")
		}
		err = cmdRemove(os.Args[2])

	case "update":
		name := ""
		if len(os.Args) >= 3 {
			name = os.Args[2]
		}
		err = cmdUpdate(name)

	case "package":
		if len(os.Args) < 3 {
			fatalf("usage: modmanager package <name>\n")
		}
		err = cmdPackage(os.Args[2])

	default:
		printError("unknown subcommand: %q", os.Args[1])
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
	fmt.Println("  modmanager " + yellow("<subcommand>") + " [arguments]")
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
