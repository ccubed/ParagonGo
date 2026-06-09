package scripting

import (
	"errors"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/dop251/goja"
)

// gojaVM is the goja-backed implementation of scriptVM.
type gojaVM struct {
	VM            *goja.Runtime
	callableCache map[string]goja.Callable
	cacheSize     int
	// maxCacheSize is the upper bound on cached callable lookups.
	// A value of 0 means unlimited — the cache grows to hold every function
	// name that has been looked up at least once.
	maxCacheSize int
	// loadedAt records when this VM was compiled and run. Used for hot-reload
	// comparisons against the script file's modification time.
	loadedAt time.Time
}

// gojaRegistrar adapts *goja.Runtime to the registrar interface. goja's Set
// returns an error which the shared registration helpers do not need.
type gojaRegistrar struct {
	vm *goja.Runtime
}

func (r gojaRegistrar) Set(name string, value any) {
	_ = r.vm.Set(name, value)
}

// gojaValue wraps a goja.Value to satisfy scriptValue.
type gojaValue struct {
	v goja.Value
}

func (g gojaValue) Export() any {
	if g.v == nil {
		return nil
	}
	return g.v.Export()
}

func newGojaVM(vm *goja.Runtime, cacheSize int) *gojaVM {
	return &gojaVM{VM: vm, callableCache: make(map[string]goja.Callable, cacheSize), maxCacheSize: cacheSize}
}

func (vmw *gojaVM) LoadedAt() time.Time {
	return vmw.loadedAt
}

func (vmw *gojaVM) ToValue(v any) any {
	return vmw.VM.ToValue(v)
}

func (vmw *gojaVM) GetFunction(name string) (callableFunc, bool) {

	fn, ok := vmw.callableCache[name]

	if ok {
		return fn, fn != nil
	}

	fn, ok = goja.AssertFunction(vmw.VM.Get(name))

	if vmw.maxCacheSize == 0 || vmw.cacheSize < vmw.maxCacheSize {
		vmw.cacheSize++
		vmw.callableCache[name] = fn
	}

	return fn, ok
}

func (vmw *gojaVM) Call(timeout time.Duration, fn callableFunc, args ...any) (scriptValue, error) {
	callable, ok := fn.(goja.Callable)
	if !ok || callable == nil {
		return nil, errors.New("invalid goja callable")
	}

	gojaArgs := make([]goja.Value, len(args))
	for i, a := range args {
		if gv, ok := a.(goja.Value); ok {
			gojaArgs[i] = gv
		} else {
			gojaArgs[i] = vmw.VM.ToValue(a)
		}
	}

	tmr := time.AfterFunc(timeout, func() {
		vmw.VM.Interrupt(errTimeout)
	})
	res, err := callable(goja.Undefined(), gojaArgs...)
	vmw.VM.ClearInterrupt()
	tmr.Stop()
	if err != nil {
		return nil, wrapVMError("call", err)
	}
	return gojaValue{v: res}, nil
}

// loadGojaVM compiles and runs script source in a new Goja VM, returning a ready
// scriptVM. scriptLabel is used in compile error messages (e.g. "room-42").
// afterLoad, if non-nil, runs the onLoad() hook under scriptLoadTimeout.
func loadGojaVM(scriptLabel string, source string, afterLoad func(scriptVM) error) (scriptVM, error) {
	vm := goja.New()
	setAllScriptingFunctions(gojaRegistrar{vm: vm})

	prg, err := goja.Compile(scriptLabel, source, false)
	if err != nil {
		return nil, fmt.Errorf("Compile: %w", err)
	}

	tmr := time.AfterFunc(scriptLoadTimeout, func() {
		vm.Interrupt(errTimeout)
	})
	_, err = vm.RunProgram(prg)
	vm.ClearInterrupt()
	tmr.Stop()
	if err != nil {
		return nil, wrapVMError("RunProgram", err)
	}

	vmw := newGojaVM(vm, 0)
	vmw.loadedAt = time.Now()

	if afterLoad != nil {
		tmr = time.AfterFunc(scriptLoadTimeout, func() {
			vm.Interrupt(errTimeout)
		})
		err = afterLoad(vmw)
		vm.ClearInterrupt()
		tmr.Stop()
		if err != nil {
			return nil, wrapVMError("onLoad", err)
		}
	}

	return vmw, nil
}

// loadVM compiles and runs script source in a new VM of the appropriate engine,
// chosen by src.Lang. afterLoad, if non-nil, is invoked with the ready VM to run
// any onLoad() hook.
func loadVM(scriptLabel string, src ScriptSource, afterLoad func(scriptVM) error) (scriptVM, error) {
	switch src.Lang {
	case LangLua:
		return loadLuaVM(scriptLabel, src.Source, afterLoad)
	case LangJS:
		return loadGojaVM(scriptLabel, src.Source, afterLoad)
	default:
		return nil, errNoScript
	}
}

// wrapVMError wraps a VM execution error with a context prefix and logs it.
func wrapVMError(context string, err error) error {
	finalErr := fmt.Errorf("%s: %w", context, err)
	if _, ok := finalErr.(*goja.Exception); ok {
		mudlog.Error("SCRIPTVM", "exception", finalErr)
	} else if errors.Is(finalErr, errTimeout) {
		mudlog.Error("SCRIPTVM", "interrupted", finalErr)
	} else {
		mudlog.Error("SCRIPTVM", "error", finalErr)
	}
	return finalErr
}

// runCallable executes fn under timeout via the VM and returns the result.
func runCallable(vmw scriptVM, timeout time.Duration, fn callableFunc, args ...any) (scriptValue, error) {
	return vmw.Call(timeout, fn, args...)
}

// validateGojaScript compiles JavaScript source without running it.
func validateGojaScript(label string, script string) ValidationResult {
	_, err := goja.Compile(label, script, false)
	if err != nil {
		if synErr, ok := err.(*goja.CompilerSyntaxError); ok {
			result := ValidationResult{
				Valid: false,
				Error: synErr.Message,
			}
			if synErr.File != nil {
				pos := synErr.File.Position(synErr.Offset)
				result.Line = pos.Line
				result.Column = pos.Column
			}
			return result
		}
		return ValidationResult{
			Valid: false,
			Error: err.Error(),
		}
	}
	return ValidationResult{Valid: true}
}
