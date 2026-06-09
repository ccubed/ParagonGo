package scripting

import (
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

// installLuaIterators replaces the global pairs/ipairs so they transparently
// iterate values that gopher-luar exposes as userdata (Go maps and slices) in
// addition to native Lua tables.
//
// Without this, Lua scripts that iterate a value returned from Go (temp data,
// GetPlayers(), UtilFindMatchIn(), etc.) fail with "table expected, got
// userdata", whereas the JavaScript engine iterates those values transparently
// with for...in / for...of. Both replacements convert luar userdata into a
// native Lua table and then defer to the built-in iterator.
func installLuaIterators(L *lua.LState) {
	builtinPairs := L.GetGlobal("pairs")
	builtinIPairs := L.GetGlobal("ipairs")

	L.SetGlobal("pairs", L.NewFunction(func(L *lua.LState) int {
		return iterate(L, builtinPairs)
	}))
	L.SetGlobal("ipairs", L.NewFunction(func(L *lua.LState) int {
		return iterate(L, builtinIPairs)
	}))
}

// iterate normalizes argument #1 to a native Lua table (when it is luar
// userdata wrapping a Go map/slice) and then tail-calls the supplied built-in
// iterator (pairs or ipairs).
func iterate(L *lua.LState, builtin lua.LValue) int {
	val := L.CheckAny(1)

	if tbl := luarToTable(L, val); tbl != nil {
		val = tbl
	}

	if err := L.CallByParam(lua.P{Fn: builtin, NRet: 3, Protect: true}, val); err != nil {
		L.RaiseError("%s", err.Error())
		return 0
	}
	return 3
}

// luarToTable converts a luar userdata wrapping a Go map, slice, or array into a
// native Lua table. It returns nil for any other value (including native
// tables), signalling that no conversion was needed.
func luarToTable(L *lua.LState, v lua.LValue) *lua.LTable {
	ud, ok := v.(*lua.LUserData)
	if !ok || ud.Value == nil {
		return nil
	}

	rv := reflect.ValueOf(ud.Value)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	tbl := L.NewTable()
	switch rv.Kind() {
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			tbl.RawSet(goToLua(L, k.Interface()), goToLua(L, rv.MapIndex(k).Interface()))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			tbl.RawSetInt(i+1, goToLua(L, rv.Index(i).Interface()))
		}
	default:
		return nil
	}
	return tbl
}

// goToLua converts a plain Go value into a Lua value, recursing into nested
// maps and slices so the whole structure is iterable. Non-container values use
// gopher-luar's standard conversion.
func goToLua(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case string:
		return lua.LString(val)
	case int:
		return lua.LNumber(val)
	case int64:
		return lua.LNumber(val)
	case uint64:
		return lua.LNumber(val)
	case float64:
		return lua.LNumber(val)
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		if tbl := luarToTable(L, &lua.LUserData{Value: v}); tbl != nil {
			return tbl
		}
	}
	return luarNew(L, v)
}
