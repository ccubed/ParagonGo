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
			fatalf("usage: modmanager install <name>\n")
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
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
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
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Print(`GoMud Module Manager

Usage:
  modmanager <subcommand> [arguments]

Subcommands:
  list                  List available modules from the registry
  info    <name>        Show details for a module
  install <name>        Download, verify, and install a module
  remove  <name>        Remove an installed module
  update  [name]        Check for updates; update a specific module if name given
  package <name>        Package a local module into a .tar.gz and print its SHA256

Run without arguments (with a terminal) to start interactive mode.

After installing or removing a module, rebuild the server:
  make build
  (or: go generate && go build -o go-mud-server)

Registry: https://raw.githubusercontent.com/GoMudEngine/GoMud-Modules/refs/heads/master/module-registry.yaml
`)
}
