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
	fmt.Println()
	fmt.Println(cyan("GoMud Module Manager") + gray(" (interactive)"))
	fmt.Println(gray("Type 'help' for a list of commands, 'quit' to exit."))
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(cyan(">") + " ")
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
				printError("%v", err)
			}

		case "info":
			if len(args) < 1 {
				printError("usage: info <name>")
				continue
			}
			if err := cmdInfo(args[0]); err != nil {
				printError("%v", err)
			}

		case "install":
			if len(args) < 1 {
				printError("usage: install <name|all-official>")
				continue
			}
			if err := cmdInstall(args[0]); err != nil {
				printError("%v", err)
			}

		case "remove":
			if len(args) < 1 {
				printError("usage: remove <name>")
				continue
			}
			if err := cmdRemove(args[0]); err != nil {
				printError("%v", err)
			}

		case "update":
			name := ""
			if len(args) >= 1 {
				name = args[0]
			}
			if err := cmdUpdate(name); err != nil {
				printError("%v", err)
			}

		case "package":
			if len(args) < 1 {
				printError("usage: package <name>")
				continue
			}
			if err := cmdPackage(args[0]); err != nil {
				printError("%v", err)
			}

		default:
			printError("unknown command: %q (type 'help' for a list)", cmd)
		}
	}
}

// isInteractiveTerminal reports whether stdin is an interactive terminal.
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func printInteractiveHelp() {
	fmt.Println()
	fmt.Println(bold("Commands:"))
	type entry struct{ cmd, desc string }
	entries := []entry{
		{green("list"), "List available modules from the registry"},
		{green("info") + " <name>", "Show details for a module"},
		{green("install") + " <name>", "Download, verify, and install a module"},
		{green("install") + " all-official", "Install all official GoMud modules at once"},
		{green("remove") + " <name>", "Remove an installed module"},
		{green("update") + " [name]", "Check for updates; update a specific module if name given"},
		{green("package") + " <name>", "Package a local module into a .tar.gz and print its SHA256"},
		{green("help"), "Show this help"},
		{green("quit") + " / " + green("exit"), "Exit the module manager"},
	}
	for _, e := range entries {
		fmt.Printf("  %s  %s\n", padRight(e.cmd, 22), e.desc)
	}
	fmt.Println()
	fmt.Println(bold("After installing or removing a module, rebuild the server:"))
	fmt.Println(codeSnippet("make build"))
	fmt.Println()
}
