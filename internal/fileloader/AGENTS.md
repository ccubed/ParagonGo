# File Loader Guide

## Scope

- Use this file for YAML-backed load/save helpers, path validation, and batch file operations in `internal/fileloader`.
- This package sits underneath many data-backed systems.

## Working Rules

- Preserve path validation and file-location consistency checks unless the task explicitly changes the persistence contract.
- Be careful with save behavior. Changes to careful-save or batch-save logic can affect data safety across the repo.
- Keep the generic loader interfaces stable unless multiple real callers are updated together.
- Prefer fixing caller-specific assumptions in the caller when possible instead of weakening loader guarantees globally.

## Verification

- Run targeted package tests for load/save, validation, or concurrent batch behavior changes.
- If a change affects how files are laid out or written, verify both successful load and successful save paths.
- Use a higher-level caller test when the change affects a real subsystem's persistence flow.

## Documentation

- Keep this file focused on persistence guarantees and loader guardrails.
