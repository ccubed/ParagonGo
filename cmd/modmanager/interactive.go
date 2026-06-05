package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// runInteractive starts an interactive REPL session. It is only called when
// the binary is invoked with no arguments and stdin is a terminal.
func runInteractive() {
	fmt.Print(`GoMud Module Manager (interactive)
Type 'help' for a list of commands, 'quit' to exit.

`)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			// EOF (Ctrl-D)
			fmt.Println()
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "quit", "exit":
			return

		case "help":
			printInteractiveHelp()

		case "list":
			if err := cmdList(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "info":
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "usage: info <name>")
				continue
			}
			if err := cmdInfo(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "install":
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "usage: install <name>")
				continue
			}
			if err := cmdInstall(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "remove":
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "usage: remove <name>")
				continue
			}
			if err := cmdRemove(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "update":
			name := ""
			if len(args) >= 1 {
				name = args[0]
			}
			if err := cmdUpdate(name); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		case "package":
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "usage: package <name>")
				continue
			}
			if err := cmdPackage(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}

		default:
			fmt.Fprintf(os.Stderr, "unknown command: %q (type 'help' for a list)\n", cmd)
		}
	}
}

// isInteractiveTerminal reports whether stdin is an interactive terminal.
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func printInteractiveHelp() {
	fmt.Print(`Commands:
  list                  List available modules from the registry
  info    <name>        Show details for a module
  install <name>        Download, verify, and install a module
  remove  <name>        Remove an installed module
  update  [name]        Check for updates; update a specific module if name given
  package <name>        Package a local module into a .tar.gz and print its SHA256
  help                  Show this help
  quit / exit           Exit the module manager

After installing or removing a module, rebuild the server:
  make build

`)
}
