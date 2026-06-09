# Quests Guide

## Scope

- `internal/quests` owns quest definitions, token-based progression, file loading, and admin persistence.
- The package is intentionally small; most quest behavior is data-driven through YAML under the data files tree.

## Files

- `quests.go`: quest structs, token parsing, progression checks, cache access, and bulk loading.
- `admin.go`: save/delete helpers that validate quest data and update the in-memory registry.
- `plugin.go`: module data-file integration. `RegisterFS(...)` registers plugin filesystems; `loadPluginQuests` merges embedded `quests/*.yaml` into the registry inside `LoadDataFiles`. Disk quests win on duplicate ids.

## Working Rules

- Keep token semantics stable. `PartsToToken`, `TokenToParts`, and `IsTokenAfter` must agree on how single-step quests and `"start"`/`"end"` behave.
- Quest files are keyed by `QuestId`; avoid changes that let filenames drift from IDs or names without updating admin persistence too.
- `GetQuest` validates the requested step against the loaded quest. If you change step rules, update both lookup and progression checks together.
- This package is a registry layer, not the reward executor. Do not move character, item, or teleport side effects here unless the rest of the repo already expects that boundary to change.

## Verification

- Run `go test ./internal/quests`.
- If you change token or persistence behavior, verify at least one load/save path and one multi-step progression path.
