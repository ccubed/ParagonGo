# Buffs Guide

## Scope

- `internal/buffs` defines buff specs, active buff instances, flag helpers, and admin save/load helpers.
- Treat this package as the core status-effect engine used by spells, items, races, and room effects.

## Files

- `buffspec.go`: immutable buff definitions, validation, duration math, and data-file loading.
- `buffs.go`: runtime buff state, collection indexing, trigger flow, pruning, and stat aggregation.
- `flags.go`: shared flag constants used by other systems.
- `admin.go`: CRUD helpers that persist buff YAML and refresh in-memory state.
- `plugin.go`: module data-file integration. `RegisterFS(...)` registers plugin filesystems; `loadPluginBuffs` merges embedded `buffs/*.yaml` into the spec map inside `LoadDataFiles` (after disk load, before items load). `RegisterBuffScript(buffId, src)` registers embedded JS; `BuffSpec.GetScript()` checks it before the disk path. Disk buffs win on duplicate ids.

## Working Rules

- Keep spec fields and runtime behavior aligned. If you add a field to `BuffSpec`, update validation, load/save paths, and any admin helpers in the same change.
- Preserve the `buffIds` and `buffFlags` indexes in `Buffs`; changes to add/remove/expire behavior usually need `Validate()` or equivalent rebuild logic to stay correct.
- `TriggerCount`, `TriggerRate`, and permanent buffs interact. Check both one-shot and repeating behavior before changing expiration logic.
- Flag changes are cross-cutting. Search for the flag name outside this package before renaming or changing semantics.

## Verification

- Prefer targeted Go tests for this package first: `go test ./internal/buffs`.
- If buff behavior changes affect combat, items, or scripting flows, run broader validation before claiming safety.
