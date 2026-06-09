package scripting

import (
	"testing"
	"time"
)

// loadEngine is a small test helper that builds a VM for the given language and
// source, with no onLoad hook.
func loadEngine(t *testing.T, lang ScriptLang, source string) scriptVM {
	t.Helper()
	vm, err := loadVM("test", ScriptSource{Lang: lang, Source: source}, nil)
	if err != nil {
		t.Fatalf("loadVM(%v) failed: %v", lang, err)
	}
	return vm
}

func TestEngines_BoolReturnAndArgs(t *testing.T) {
	cases := []struct {
		name   string
		lang   ScriptLang
		source string
	}{
		{
			name:   "js",
			lang:   LangJS,
			source: `function onCommand(rest) { return rest === "yes"; }`,
		},
		{
			name:   "lua",
			lang:   LangLua,
			source: `function onCommand(rest) return rest == "yes" end`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := loadEngine(t, tc.lang, tc.source)

			fn, ok := vm.GetFunction("onCommand")
			if !ok {
				t.Fatalf("onCommand not found")
			}

			res, err := vm.Call(time.Second, fn, vm.ToValue("yes"))
			if err != nil {
				t.Fatalf("Call returned error: %v", err)
			}
			if b, ok := res.Export().(bool); !ok || !b {
				t.Fatalf("expected true, got %#v", res.Export())
			}

			res, err = vm.Call(time.Second, fn, vm.ToValue("no"))
			if err != nil {
				t.Fatalf("Call returned error: %v", err)
			}
			if b, ok := res.Export().(bool); !ok || b {
				t.Fatalf("expected false, got %#v", res.Export())
			}
		})
	}
}

// counter is a simple Go value with an exported method, used to confirm that
// both engines reflect Go methods and that calls mutate the underlying pointer.
type counter struct {
	N int
}

func (c *counter) Add(n int) { c.N += n }

func TestEngines_MethodReflectionMutatesGo(t *testing.T) {
	cases := []struct {
		name   string
		lang   ScriptLang
		source string
	}{
		{
			name:   "js",
			lang:   LangJS,
			source: `function run(c) { c.Add(5); }`,
		},
		{
			name:   "lua",
			lang:   LangLua,
			source: `function run(c) c:Add(5) end`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := loadEngine(t, tc.lang, tc.source)

			fn, ok := vm.GetFunction("run")
			if !ok {
				t.Fatalf("run not found")
			}

			c := &counter{}
			if _, err := vm.Call(time.Second, fn, vm.ToValue(c)); err != nil {
				t.Fatalf("Call returned error: %v", err)
			}
			if c.N != 5 {
				t.Fatalf("expected counter mutated to 5, got %d", c.N)
			}
		})
	}
}

func TestEngines_GlobalGoFunction(t *testing.T) {
	// RandInt is bound as a global by setUtilFunctions for both engines.
	cases := []struct {
		name   string
		lang   ScriptLang
		source string
	}{
		{
			name:   "js",
			lang:   LangJS,
			source: `function run() { return RandInt(5, 5); }`,
		},
		{
			name:   "lua",
			lang:   LangLua,
			source: `function run() return RandInt(5, 5) end`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vm := loadEngine(t, tc.lang, tc.source)

			fn, ok := vm.GetFunction("run")
			if !ok {
				t.Fatalf("run not found")
			}

			res, err := vm.Call(time.Second, fn)
			if err != nil {
				t.Fatalf("Call returned error: %v", err)
			}
			// goja returns int64, lua returns float64; compare numerically.
			switch v := res.Export().(type) {
			case int64:
				if v != 5 {
					t.Fatalf("expected 5, got %d", v)
				}
			case float64:
				if v != 5 {
					t.Fatalf("expected 5, got %f", v)
				}
			default:
				t.Fatalf("unexpected return type %T (%#v)", v, v)
			}
		})
	}
}

func TestValidateScript_Lua(t *testing.T) {
	if r := ValidateScript("test", `function ok() return true end`, LangLua); !r.Valid {
		t.Fatalf("expected valid lua, got error: %s", r.Error)
	}
	if r := ValidateScript("test", `function bad( return end`, LangLua); r.Valid {
		t.Fatalf("expected invalid lua to be reported")
	}
}

func TestLangFromPath(t *testing.T) {
	if LangFromPath("foo.js") != LangJS {
		t.Fatalf("expected js")
	}
	if LangFromPath("foo.lua") != LangLua {
		t.Fatalf("expected lua")
	}
	if LangFromPath("foo.yaml") != LangNone {
		t.Fatalf("expected none")
	}
}
