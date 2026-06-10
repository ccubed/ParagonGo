// Package modmanager implements the GoMud community module manager.
// It is invoked via the main binary with: go-mud-server module [subcommand] [args]
package modmanager

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Run is the entry point for the module manager subcommand. args should be
// os.Args[2:] (everything after "module").
func Run(args []string) {
	args, err := applyManifestFlag(args)
	if err != nil {
		fatalf("%v\n", err)
	}

	if len(args) == 0 {
		if isInteractiveTerminal() {
			runInteractive()
			return
		}
		printUsage()
		os.Exit(0)
	}

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

// applyManifestFlag scans args for a global --manifest <source> (or
// --manifest=<source>) flag, applies it as the manifest override, and returns
// args with the flag and its value removed so subcommand parsing is unaffected.
// The flag may appear anywhere in args.
func applyManifestFlag(args []string) ([]string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--manifest":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--manifest requires a file path or URL")
			}
			if err := useManifestOverride(args[i+1]); err != nil {
				return nil, err
			}
			i++ // skip the value we just consumed
		case strings.HasPrefix(a, "--manifest="):
			if err := useManifestOverride(strings.TrimPrefix(a, "--manifest=")); err != nil {
				return nil, err
			}
		default:
			rest = append(rest, a)
		}
	}
	return rest, nil
}

// useManifestOverride validates source, points the module manager at it for the
// remainder of this invocation, and warns the user that the default registry is
// being overridden. The special identifiers "default"/"reset" and "test" are
// resolved to the built-in registry URLs before validation.
func useManifestOverride(source string) error {
	resolved := resolveManifestSource(source)
	if err := validateManifestSource(resolved); err != nil {
		return err
	}
	manifestSource = resolved
	if resolved != registryURL {
		printManifestOverrideWarning(resolved)
	}
	return nil
}

// resolveManifestSource maps the special identifiers "default"/"reset" to the
// official registry URL and "test" to the test registry URL. Any other value is
// returned unchanged so it can be treated as a literal file path or URL.
func resolveManifestSource(source string) string {
	switch source {
	case "default", "reset":
		return registryURL
	case "test":
		return registryTestURL
	default:
		return source
	}
}

// validateManifestSource ensures a custom manifest location points at a YAML
// file. Both http(s) URLs and local filesystem paths are accepted, as long as
// they end in .yaml or .yml (any query string or fragment on a URL is ignored).
func validateManifestSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("--manifest requires a file path or URL")
	}
	lower := strings.ToLower(source)
	if i := strings.IndexAny(lower, "?#"); i >= 0 {
		lower = lower[:i]
	}
	if !strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(lower, ".yml") {
		return fmt.Errorf("manifest must be a .yaml file, got %q", source)
	}
	return nil
}

// printManifestOverrideWarning warns that a non-default manifest is in use.
func printManifestOverrideWarning(source string) {
	printWarning("using a custom module manifest instead of the default registry: %s", bold(source))
	fmt.Fprintln(os.Stderr, gray("         only use manifests from sources you trust; downloads are still SHA256-verified."))
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
	fmt.Println(bold("Global options:"))
	fmt.Printf("  %s  %s\n", padRight(green("--manifest")+" <path|url>", 22),
		"Temporarily use an alternate module manifest (.yaml file or")
	fmt.Printf("  %s  %s\n", padRight("", 22),
		"local path) instead of the default registry; for local testing.")
	fmt.Printf("  %s  %s\n", padRight("", 22),
		"Use 'test' for the built-in test registry, 'default' to reset.")
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
