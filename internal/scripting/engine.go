package scripting

import (
	"strings"
	"time"
)

// ScriptLang identifies the scripting language a script source is written in.
type ScriptLang uint8

const (
	LangNone ScriptLang = iota
	LangJS
	LangLua
)

// LangFromPath returns the ScriptLang implied by a script file extension.
// Unknown extensions return LangNone.
func LangFromPath(path string) ScriptLang {
	switch {
	case strings.HasSuffix(path, `.js`):
		return LangJS
	case strings.HasSuffix(path, `.lua`):
		return LangLua
	default:
		return LangNone
	}
}

// ScriptSource carries a resolved script's source code along with the language
// it is written in (derived from its file extension) and the path it was loaded
// from (used for hot-reload mtime checks).
type ScriptSource struct {
	Lang   ScriptLang
	Source string
	Path   string
}

// sourceFromPath builds a ScriptSource from a resolved script path and its
// source string, inferring the language from the file extension.
func sourceFromPath(path string, source string) ScriptSource {
	if len(source) == 0 {
		return ScriptSource{Lang: LangNone}
	}
	return ScriptSource{Lang: LangFromPath(path), Source: source, Path: path}
}

// registrar is the minimal surface needed to expose Go functions and values to
// a script VM as named globals. Both the goja and Lua engines implement it so
// the shared set*Functions() helpers can target either engine unchanged.
type registrar interface {
	// Set binds value (a Go function, struct, slice, map, etc.) to a global
	// name inside the VM.
	Set(name string, value any)
}

// scriptValue is the engine-neutral wrapper around a value returned from a
// script call. It exposes only what the Try* event functions need.
type scriptValue interface {
	// Export returns the value as a plain Go value (bool, string, etc.).
	Export() any
}

// scriptVM is the engine-neutral interface used by the cache and event-dispatch
// code. Both gojaVM and luaVM implement it.
type scriptVM interface {
	// GetFunction looks up a global function by name. The second return is
	// false when no callable with that name exists.
	GetFunction(name string) (callableFunc, bool)
	// ToValue converts a Go value into an engine value suitable for passing as
	// a call argument.
	ToValue(v any) any
	// Call invokes fn under the supplied timeout with the given arguments.
	// Arguments are raw Go values or values previously produced by ToValue.
	Call(timeout time.Duration, fn callableFunc, args ...any) (scriptValue, error)
	// LoadedAt reports when the VM was compiled and run, for hot-reload checks.
	LoadedAt() time.Time
}

// callableFunc is an opaque handle to a script function resolved from a VM.
// Its concrete type depends on the engine; callers pass it back to scriptVM.Call.
type callableFunc any
