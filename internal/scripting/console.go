package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// consoleObject returns the global `console` object exposed to scripts as a map
// of named functions. Both engines reflect a map[string]any into an indexable
// object with the same keys, so scripts call console.log(...) identically in
// JavaScript and Lua.
func consoleObject() map[string]any {
	return map[string]any{
		`log`: func(msg any) {
			mudlog.Info(`SCRIPTVM`, `msg`, msg)
		},
		`info`: func(msg any) {
			mudlog.Info(`SCRIPTVM`, `msg`, msg)
		},
		`debug`: func(msg any) {
			mudlog.Debug(`SCRIPTVM`, `msg`, msg)
		},
		`warn`: func(msg any) {
			mudlog.Warn(`SCRIPTVM`, `msg`, msg)
		},
		`error`: func(msg any) {
			mudlog.Error(`SCRIPTVM`, `msg`, msg)
		},
	}
}
