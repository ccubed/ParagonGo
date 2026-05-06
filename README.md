# GoMud

## Overview

![image](feature-screenshots/splash.png)

**GoMud** is an open-source MUD (_Multi-User Dungeon_) game server and library, written in Go.

It includes a fully playable default world, and provides built-in tools to customize or create your own.

Playable online demo: **<http://www.gomud.net>**

---

<!-- TOC -->
- [GoMud](#gomud)
  - [Overview](#overview)
  - [Features](#features)
    - [Screenshots](#screenshots)
    - [ANSI Colors](#ansi-colors)
    - [Small Feature Demos](#small-feature-demos)
  - [Setup](#setup)
    - [Requirements](#requirements)
    - [Quick Start](#quick-start)
  - [Connecting](#connecting)
  - [Common Server Commands](#common-server-commands)
  - [Configuration](#configuration)
    - [Config Files](#config-files)
    - [Enable Server HTTPS Support](#enable-server-https-support)
  - [User Support](#user-support)
  - [Development Notes](#development-notes)
    - [Contributor Guide](#contributor-guide)
    - [Build Commands](#build-commands)
    - [Env Vars](#env-vars)
    - [Why Go?](#why-go)

---

<!-- /TOC -->

## Features

### Screenshots

Click below to see in-game screenshots of just a handful of features:

[![Feature Screenshots](feature-screenshots/screenshots-thumb.png "Feature Screenshots")](feature-screenshots/README.md)

### ANSI Colors

Colorization is handled through extensive use of my [github.com/GoMudEngine/ansitags](https://github.com/GoMudEngine/ansitags) library.

### Small Feature Demos

- [Web Admin Tool](https://youtu.be/n44kQp2JwIk)
- [Web Map Editor](https://youtu.be/W2F07TeR168)
- [Auto-complete input](https://youtu.be/7sG-FFHdhtI)
- [In-game maps](https://youtu.be/navCCH-mz_8)
- [Quests / Quest Progress](https://youtu.be/3zIClk3ewTU)
- [Lockpicking](https://youtu.be/-zgw99oI0XY)
- [Hired Mercs](https://youtu.be/semi97yokZE)
- [TinyMap](https://www.youtube.com/watch?v=VLNF5oM4pWw) (okay not much of a "feature")
- [256 Color/xterm](https://www.youtube.com/watch?v=gGSrLwdVZZQ)
- [Customizable Prompts](https://www.youtube.com/watch?v=MFkmjSTL0Ds)
- [Mob/NPC Scripting](https://www.youtube.com/watch?v=li2k1N4p74o)
- [Room Scripting](https://www.youtube.com/watch?v=n1qNUjhyOqg)
- [Kill Stats](https://www.youtube.com/watch?v=4aXs8JNj5Cc)
- [Searchable Inventory](https://www.youtube.com/watch?v=iDUbdeR2BUg)
- [Day/Night Cycles](https://www.youtube.com/watch?v=CiEbOp244cw)
- [Web Socket "Virtual Terminal"](https://www.youtube.com/watch?v=L-qtybXO4aw)
- [Alternate Characters](https://www.youtube.com/watch?v=VERF2l70W34)

---

## Setup

### Requirements

- `go` 1.24 or newer
- Optional: `docker` for container builds/test/runs

### Quick Start

In a Terminal, run the following commands:

```shell
git clone https://github.com/GoMudEngine/GoMud.git
cd GoMud

make reset-admin-pw   # set a new default admin password
make run              # runs GoMud server using `go`

make docker-run       # Alternatively, run the GoMud server using `docker`
```

Then open your browser to: `http://localhost`

---

## Connecting

When the GoMud server is running, you can connect it via the Terminal, or with a web browser.

- Telnet: `localhost:33333` or `localhost:44444`
- Local-only telnet port: `127.0.0.1:9999`

- Web client: [http://localhost/webclient](http://localhost/webclient)
- Web admin: [http://localhost/admin/](http://localhost/admin/)

**Important:** Run `make reset-admin-pw`, otherwise your default world will launch with these credentials:

- Username: `admin`
- Password: `password`

## Common Server Commands

In a Terminal, run one of the following commands:

```shell

make run          # runs GoMud using the `go` framework

make build        # creates a executable binary of GoMud at `./go-mud-server`

make run-docker   # runs GoMud in a container using Docker Compose

make help         # shows all available `make` command options
```

## Configuration

### Config Files

GoMud loads configuration in layers so you can keep your own world-specific changes separate from the bundled defaults:

```text
_datafiles/config.yaml
  -> FilePaths.DataFiles (defaults to _datafiles/world/default)
      -> {DataFiles}/config-overrides.yaml
          -> environment variables such as CONFIG_PATH, LOG_PATH, LOG_LEVEL, LOG_NOCOLOR
```

- `_datafiles/config.yaml` is the bundled base config that ships with the repo, and shouldn't be edited or changed.
- `FilePaths.DataFiles` points at the active world data directory. By default that is `_datafiles/world/default`.
- `{DataFiles}/config-overrides.yaml` is the normal place to save local overrides for a world.
- `CONFIG_PATH=/path/to/config.yaml` can point GoMud at a different override file when you want to keep it outside the repo or maintain separate deploy-specific settings.

- For upgrades, treat `_datafiles/config.yaml` as a reference file, not your day-to-day edit target. 
= Keep your custom changes in `config-overrides.yaml` or a separate file selected with `CONFIG_PATH` so pulling new code does not overwrite your local settings.

### Enable Server HTTPS Support

GoMud can serve HTTPS when you provide a certificate and private key, or can be automated using LetsEncrypt provisioning.

For a guided HTTPS setup process, run:

```shell
make https-setup
```

When the admin interface is enabled, `/admin/https/` shows the current HTTPS mode, the checks GoMud ran, and the next steps needed to finish setup.

---

## User Support

If you have comments, questions, suggestions (don't be shy, your questions or requests might help others too):

- [Github Discussions](https://github.com/GoMudEngine/GoMud/discussions)

- [Discord Server](https://discord.gg/cjukKvQWyy)

- [Community Guides](_datafiles/guides/README.md)

---

## Development Notes

### Contributor Guide

Interested in contributing? Check out our [CONTRIBUTING.md](https://github.com/GoMudEngine/GoMud/blob/master/.github/CONTRIBUTING.md) to learn about the process.

### Build Commands

| Command            | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `make build`       | Validates and builds the server binary.                                     |
| `make run`         | Generates module imports and starts the server with `go run .`.             |
| `make run-new`     | Deletes generated room instance data, then starts the server fresh.         |
| `make run-docker`  | Builds and starts the server container from `compose.yml`.                  |
| `make https-setup` | Runs the interactive HTTPS certificate setup helper.                        |
| `make reset-admin-pw` | Interactively resets the admin user's password.                          |
| `make test`        | Runs code generation, JavaScript linting, and `go test -race ./...`.        |
| `make validate`    | Runs `fmtcheck` and `go vet`.                                               |
| `make ci-local`    | Builds the local CI container and runs workflow validation.                 |
| `make help`        | Lists the available developer targets.                                      |

### Env Vars

When running, several environment variables can be set to alter behaviors of the mud:

| Variable      | Example Value                   | Descripton                           |
|---------------|---------------------------------|--------------------------------------|
| `CONFIG_PATH` | `/path/to/config.yaml`          | Use alternate config file            |
| `LOG_PATH`    | `/path/to/log.txt`              | Log to file instead of stderr        |
| `LOG_LEVEL`   | `LOW` / `MEDIUM` / `HIGH`       | Set verbosity (rotates at 100MB)     |
| `LOG_NOCOLOR` | `1`                             | Disable colored log output           |

### Why Go?

Why not?

Go provides a number of practical benefits:

- **Compatible**: Easily builds across platforms and CPU architectures (Windows, Linux, MacOS, etc).
- **Fast**: Execution and build times are quick, and GoMud builds in just a couple of seconds.
- **Opinionated**: Consistent style/patterns make it easy to jump into any Go project.
- **Modern**: A relatively new language without decades of accumulated baggage.
- **Upgradable**: Strong backward compatibility makes version upgrades simple and low-risk.
- **Statically linked**: Built binaries have no dependency headaches.
- **No central registries**: Dependencies are pulled directly from source repositories.
- **Concurrent**: Concurrency is built into the language itself, not bolted on via external libraries.
