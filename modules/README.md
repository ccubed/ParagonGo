# Modules

Extract any modules into this folder.

* Modules should be named uniquely, in a manner that identifies their purpose.
* Modules should be inside of a subfolder of `modules`, named after their package.
  * Example: `modules/birds/` would contain the `birds` module/package.

* Module folders should container a `datafiles` folder that contains any datafiles needed.
  * Files within `datafiles` will be treated as though located within the actual `_datafiles`
  * These files are read-only.

## Things Modules can do:

* Access core GoMud code.
* Listen for, handle and/or cancel events (See `modules/auctions`)
  * For example, run custom code every `NewRound{}` event, or do something whenever a `LevelUp{}` event is fired.
* Handle Telnet IAC commands (See `modules/gmcp`)
* Add a handler for new connections (See `modules/gmcp`)
* Add web pages to default web site (See `modules/leaderboards`)
  * Web page template with custom data
  * (optional) Add navigation links
  * (optional) provide any downloadable/linkable assets (images, files, etc)
* Add/Over-write existing template files (See `modules/auctions`)
* Add/Over-write help files (See `modules/auctions`)
* Add/Over-write user or mob commands  (See `modules/auctions`)
* Add functions for scripting (See `modules/follow`)
* Save/Load their own data (See `modules/leaderboards`)
* Track their own config values (See `modules/leaderboards`)
* Modify help menu items, command aliases, help aliases  (See `modules/leaderboards`)

# Examples

## Basic user command function

* time/time.go
* time/files/*

## User command with maintained state and save/loading of data

* leaderboards/leaderboards.go
* leaderboards/files/*

---

# Community Module Manager

The module manager is built into the server binary. Run it via the `module`
subcommand:

## Commands

```sh
# Run the module manager interactively
go run . module 
```

Or run commands individually:

```sh

# List all available modules (fetched live from the registry)
go run . module list

# Show full details for a module
go run . module info <name>

# Install a module
go run . module install <name>

# Remove an installed module
go run . module remove <name>

# Check for updates (all installed modules)
go run . module update

# Update a specific module
go run . module update <name>
```

With a built binary:

```sh
./go-mud-server module list
./go-mud-server module install <name>
```

A `make module` shortcut is also available.

## After installing or removing a module

Modules are compiled into the server binary, so a rebuild is required for
any change to take effect:

```sh
make build
# or: go generate && go build -o go-mud-server
```

If a newly installed module imports a Go package not already in `go.mod`,
run `go mod tidy` before building.

## modules.lock.yaml

When a community module is installed, the manager writes
`modules/modules.lock.yaml` to record what is installed, at what version,
and from where. This file is managed automatically - do not edit it by hand.

You can commit `modules.lock.yaml` to source control if you want to track
which community modules your server uses. It is not required.

## Registry

The registry is defined in `module-registry.yaml` at the repo root and is
fetched from:

    https://raw.githubusercontent.com/GoMudEngine/GoMud-Modules/refs/heads/master/module-registry.yaml

To submit a new community module, open a pull request that adds an entry to
`module-registry.yaml`. Each entry requires:

- `name` - the directory name that will be created under `modules/`
- `description` - a one-line description
- `version` - semver string
- `author` - your name or organisation
- `url` - direct download link to a `.tar.gz` or `.zip` source archive
- `sha256` - SHA256 hex digest of the archive (used for integrity verification)

To generate the SHA256 of your archive:

```sh
# Linux / macOS
sha256sum your-module-1.0.0.tar.gz

# Windows (PowerShell)
Get-FileHash your-module-1.0.0.zip -Algorithm SHA256
```

## Authoring a community module

A community module is a directory of Go source files. The archive you publish
must extract to the following layout (flat or inside a single top-level
wrapper directory, which is stripped automatically):

```
<module-name>.go          <- at least one .go file; registers via init()
files/                    <- optional: embedded data files
  datafiles/
    templates/...
    html/...
  data-overlays/
    config.yaml
    keywords.yaml
```

The Go files import GoMud internals using the standard module path:

```go
import "github.com/GoMudEngine/GoMud/internal/plugins"
```

See the existing modules in this directory for complete examples.
