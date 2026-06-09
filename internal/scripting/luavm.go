package scripting

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

// luaVM is the gopher-lua + gopher-luar backed implementation of scriptVM.
type luaVM struct {
	L             *lua.LState
	callableCache map[string]*lua.LFunction
	loadedAt      time.Time
}

// luaRegistrar adapts *lua.LState to the registrar interface, binding Go values
// as Lua globals via gopher-luar reflection.
type luaRegistrar struct {
	L *lua.LState
}

func (r luaRegistrar) Set(name string, value any) {
	r.L.SetGlobal(name, luar.New(r.L, value))
}

// luarNew exposes gopher-luar's default Go->Lua conversion to other files in
// the package.
func luarNew(L *lua.LState, value any) lua.LValue {
	return luar.New(L, value)
}

// luaValue wraps an lua.LValue to satisfy scriptValue.
type luaValue struct {
	L *lua.LState
	v lua.LValue
}

func (l luaValue) Export() any {
	return exportLuaValue(l.L, l.v)
}

// exportLuaValue converts an lua.LValue into a plain Go value, mirroring the
// subset of goja's Export() behavior the Try* event functions rely on (chiefly
// bool returns).
func exportLuaValue(L *lua.LState, v lua.LValue) any {
	switch lv := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(lv)
	case lua.LString:
		return string(lv)
	case lua.LNumber:
		return float64(lv)
	case *lua.LUserData:
		return lv.Value
	default:
		return v
	}
}

func (vm *luaVM) LoadedAt() time.Time {
	return vm.loadedAt
}

func (vm *luaVM) ToValue(v any) any {
	return luar.New(vm.L, v)
}

func (vm *luaVM) GetFunction(name string) (callableFunc, bool) {
	if fn, ok := vm.callableCache[name]; ok {
		return fn, fn != nil
	}

	var fn *lua.LFunction
	if lf, ok := vm.L.GetGlobal(name).(*lua.LFunction); ok {
		fn = lf
	}

	vm.callableCache[name] = fn
	return fn, fn != nil
}

func (vm *luaVM) Call(timeout time.Duration, fn callableFunc, args ...any) (scriptValue, error) {
	luaFn, ok := fn.(*lua.LFunction)
	if !ok || luaFn == nil {
		return nil, errors.New("invalid lua callable")
	}

	luaArgs := make([]lua.LValue, len(args))
	for i, a := range args {
		if lv, ok := a.(lua.LValue); ok {
			luaArgs[i] = lv
		} else {
			luaArgs[i] = luar.New(vm.L, a)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	vm.L.SetContext(ctx)
	defer vm.L.RemoveContext()

	err := vm.L.CallByParam(lua.P{
		Fn:      luaFn,
		NRet:    1,
		Protect: true,
	}, luaArgs...)
	if err != nil {
		if ctx.Err() != nil {
			return nil, wrapVMError("call", errTimeout)
		}
		return nil, wrapVMError("call", err)
	}

	ret := vm.L.Get(-1)
	vm.L.Pop(1)
	return luaValue{L: vm.L, v: ret}, nil
}

// loadLuaVM compiles and runs Lua script source in a new gopher-lua state,
// returning a ready scriptVM. afterLoad, if non-nil, runs the onLoad() hook
// under scriptLoadTimeout.
func loadLuaVM(scriptLabel string, source string, afterLoad func(scriptVM) error) (scriptVM, error) {
	L := lua.NewState(lua.Options{
		// SkipOpenLibs is left false so scripts have access to the standard
		// Lua base, string, table, and math libraries.
		IncludeGoStackTrace: false,
	})

	setAllScriptingFunctions(luaRegistrar{L: L})
	installLuaIterators(L)

	fn, err := L.LoadString(source)
	if err != nil {
		L.Close()
		return nil, fmt.Errorf("Compile: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), scriptLoadTimeout)
	L.SetContext(ctx)
	L.Push(fn)
	runErr := L.PCall(0, lua.MultRet, nil)
	timedOut := ctx.Err() != nil
	L.RemoveContext()
	cancel()
	if runErr != nil {
		L.Close()
		if timedOut {
			return nil, wrapVMError("RunProgram", errTimeout)
		}
		return nil, wrapVMError("RunProgram", runErr)
	}

	vm := &luaVM{
		L:             L,
		callableCache: make(map[string]*lua.LFunction),
		loadedAt:      time.Now(),
	}

	if afterLoad != nil {
		if err := afterLoad(vm); err != nil {
			L.Close()
			return nil, wrapVMError("onLoad", err)
		}
	}

	return vm, nil
}

// luaErrLineRe extracts the line number from a gopher-lua compile/load error,
// which is formatted like `<string>:12: unexpected symbol`.
var luaErrLineRe = regexp.MustCompile(`:(\d+):`)

// validateLuaScript loads Lua source without running it and reports syntax
// errors, extracting a line number from the error message when present.
func validateLuaScript(label string, script string) ValidationResult {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	_, err := L.LoadString(script)
	if err == nil {
		return ValidationResult{Valid: true}
	}

	msg := err.Error()
	result := ValidationResult{Valid: false, Error: msg}
	if m := luaErrLineRe.FindStringSubmatch(msg); len(m) == 2 {
		if line, convErr := strconv.Atoi(m[1]); convErr == nil {
			result.Line = line
		}
	}
	return result
}
