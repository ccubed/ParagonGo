# GoMud Agent Guide

## Purpose

- Use this file for durable repo-wide working rules.
- Keep it short and operational. Put detailed subsystem guidance in the nearest nested `AGENTS.md`.
- Treat `Makefile`, code, and config files as the source of truth when docs drift.

## Repo Map

- `internal/`: core engine, web, config, gameplay, and runtime packages.
- `_datafiles/`: bundled world data, web assets, config, and scripts loaded by the server.
- `modules/`: optional gameplay/web modules auto-wired through code generation.
- `cmd/generate/`: code generation tooling; do not treat it like game logic.
- `.github/`: CI, release, and automation workflows.
- `scripts/`: helper scripts. Keep shell changes compatible with both `bash` and `zsh`.
- If a task touches a subsystem with its own `AGENTS.md`, read that file before editing.

## Core Commands

- Prefer repo entrypoints over ad hoc command sequences.
- Build: `make build`
- Run locally: `make run`
- Run in Docker: `make run-docker`
- Validate Go formatting and vet: `make validate`
- Full test pass: `make test`
- JavaScript lint: `make js-lint`
- Lua lint: `make lua-lint`
- Local CI dry run: `make ci-local`
- HTTPS helper: `make https-setup`
- Reset admin password: `make reset-admin-pw`

## Working Rules

- Keep PRs small and split non-essential changes instead of bundling unrelated cleanup.
- Leave unrelated tracked or untracked files alone unless the task explicitly includes them.
- When possible, trigger code generation through existing repo commands instead of custom `go generate` sequences.
- Module config keys live under `Modules.<modulename>.*` in `_datafiles/config.yaml`; modules read them through `plug.Config.Get(...)`.
- New modules are registered via Go `init()` functions; generation refreshes `cmd/generate/module-imports.go`.
- Room tags on `rooms.Room` are the main extensibility hook for module behavior.
- If you add durable guidance for a subsystem, update the nearest relevant `AGENTS.md`, not just this root file.

## Verification

- Run the relevant existing checks for the files you changed.
- For Go changes, prefer `make validate` and targeted or full Go tests as appropriate.
- For JavaScript or web asset changes, run `make js-lint` and any targeted validation that proves the behavior.
- If a test fails, propose a solution to fix it.
- Do not claim validation you did not run.

## Documentation

- Update docs when behavior, setup, operator steps, or developer workflows materially change.
- Keep long architecture or onboarding prose in narrower subfolder `README.md` docs, not in this always-loaded file.
