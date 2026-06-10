# Scripting Package Guide

## Scope

- Use this file for the multi-engine scripting runtime in `internal/scripting`.
- This package powers room, mob, item, spell, and buff scripts and exposes engine APIs to scripts.
- Two engines run side-by-side behind one abstraction: JavaScript via goja (`github.com/dop251/goja`) and Lua via gopher-lua + gopher-luar (`github.com/yuin/gopher-lua`, `layeh.com/gopher-luar`).

## Engine selection

- Language is chosen per script by file extension. A `.js` file is preferred; a `.lua` file is used only when no `.js` exists. When neither exists the default path is `.js`.
- The choice lives in the data packages' `GetScriptPath()` via `util.ResolveScriptPath`, not in this package. The scripting loaders derive the language from the resolved path with `LangFromPath` / `sourceFromPath`.
- Plugin-registered scripts (mobs/items/buffs/pets) are treated as JavaScript.

## Engine abstraction

- `scriptVM` (engine.go) is the engine-neutral interface used by the caches and `Try*` event functions. `gojaVM` (vmwrapper.go) and `luaVM` (luavm.go) implement it.
- `registrar` is the binding surface used by every `set*Functions()` helper; the same helper registers the same Go funcs/values into either engine. Do not reintroduce a hard dependency on `*goja.Runtime` in those helpers.
- Wrapper types (`ScriptActor`, `ScriptRoom`, `ScriptItem`, etc.) and global helpers are language-agnostic Go and are shared by both engines. goja and gopher-luar both reflect their exported methods and mutate through the held pointers.
- Lua method calls use `obj:Method(args)`; JS uses `obj.Method(args)`. Field access is `obj.field` in both.

## Global user script

- `user.go` implements a single global script that runs for every player (not per-user, not per-room). It is cached as one shared VM (`userVM`) keyed by nothing; there is at most one user-script VM at a time.
- The script lives at `<DataFiles>/scripts/user.js` (or `.lua`), resolved through `userScriptPathFn` (a package var so tests can redirect it). There is no YAML data record; the path is fixed and global.
- Dispatchers: `TryUserCommand` (onCommand), `TryUserScriptEvent` (onDying/onLogin/onLogout and any simple `(user, room)` handler), `TryUserDieEvent` (onDie), `TryUserLevelEvent` (onLevel). All no-op safely when the script or the handler is absent.
- `onCommand` is fired from `usercommands.TryCommand` before mob/room scripts; returning true halts all further command processing. `onDying`/`onDie` are fired from the player drop hook and `Suicide` and abort the default behavior when they return true.
- Save/edit goes through `GetUserScript`/`SaveUserScript`/`UserScriptLang` and the admin API (`/admin/api/v1/user/script`). The editor page is `/admin/users-script`; reference docs are at `/admin/scripting-users`. `SaveUserScript` calls `InvalidateUserVM` so edits take effect without restart; `ClearUserVM` is also wired into `PruneVMs(forceClear)`.
- The editor schema entry is `userScriptType()` in `schema.go` (scriptType key `"user"`); `LevelEventDetails` is declared in the `.d.ts`.

## Editor intellisense (admin script editor)

- The admin Monaco editor is language-aware. JavaScript uses the TypeScript language service seeded by the generated `.d.ts` (`internal/web/api_v1_scripting_dts.go`). Lua has no language service, so type-aware intellisense is built in `monaco-editor-frame.js` from two JSON endpoints.
- `schema.go` (`GetScriptFunctionsSchema`, served at `/admin/api/v1/scripting/functions`) describes engine global functions and per-script-type event handler signatures.
- `objecttypes.go` (`GetScriptObjectTypes`, served at `/admin/api/v1/scripting/objecttypes`) describes the methods of the runtime object types (`ActorObject`, `RoomObject`, `ItemObject`, `PetObject`, `PartyObject`, `ContainerObject`). This is the structured source the Lua editor uses to give context-sensitive `obj:Method()` completions. When a wrapper method returns or accepts one of these object types, type it by name (e.g. `PartyObject`) rather than `any`/`object` so both editors can chain completions.
- The object-type method lists must stay in sync with the hand-authored TypeScript interfaces in `internal/web/api_v1_scripting_dts.go`. `internal/web/objecttypes_sync_test.go` compares method names and fails on drift. When you add or remove a wrapper method exposed to scripts, update the `.d.ts` interface, `objecttypes.go`, and (if it is a global) `schema.go` together.

## Working Rules

- Preserve the existing separation between script categories and their wrappers. Do not collapse room, mob, item, spell, and buff behavior into a single ad hoc path.
- Keep script-exposed APIs stable across both engines unless the task explicitly changes the scripting contract.
- Be careful with timeout, VM reuse, and wrapper behavior. goja uses `Interrupt`; Lua uses a per-call `context` deadline. Small runtime changes here can affect all scripted content.
- Prefer extending existing script helpers and wrapper methods rather than adding one-off special cases in individual call paths.
- When changing what scripts can do, consider both engines and content compatibility with existing world scripts.

## Verification

- Run targeted `internal/scripting` tests for runtime, wrapper, or helper changes. `engine_test.go` exercises both engines for bool returns, method reflection/mutation, and global Go functions.
- If the change affects a specific script surface such as room or item scripts, verify the nearest package tests and any directly affected integration path.
- Call out any compatibility risk for existing scripts if behavior changed but broad world-content testing was not run.

## Documentation

- Keep this file focused on runtime contracts and safety rules.
- Put long API catalogs in narrower docs if they are still needed.
