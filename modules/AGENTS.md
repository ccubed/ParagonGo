# Modules Guide

## Scope

- Use this file for the top-level `modules/` tree and shared module conventions.
- Individual modules may also have their own `AGENTS.md`; read the nearest one before changing a specific module.

## Working Rules

- Prefer implementing optional or feature-scoped gameplay in a module instead of modifying core packages when the feature does not need to be global.
- Module config belongs under `Modules.<modulename>.*` in `_datafiles/config.yaml`, and module code should read it via `plug.Config.Get(...)`.
- New modules are discovered through Go `init()` registration and wired through generation. After adding or removing modules, use the repo command flow that refreshes generated imports.
- If a module depends on room tags, reserve and document those tags explicitly. Room tags are the main opt-in integration point for module behavior.
- Reuse the existing plugin hooks for:
  - user or mob commands
  - event listeners
  - scripting exports
  - public web pages
  - admin pages
  - admin API endpoints
- Keep module-specific UI, files, and persistence inside the module when practical instead of spreading the feature across unrelated packages.

## Verification

- Run the narrowest tests that cover the changed module behavior, then broader repo checks if the change crosses package boundaries.
- If a module adds or changes admin or public web surfaces, verify the registered route path and backing asset/template path together.
- If a module changes generated wiring or registration, confirm the generated imports stay current through the repo entrypoint.

## Documentation

- Keep this file about module-authoring rules, not a full plugin API tutorial.
- Put reusable module-specific details in the module's own directory guide when they are not repo-wide rules.
