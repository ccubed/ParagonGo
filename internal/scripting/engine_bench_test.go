package scripting

import (
	"testing"
	"time"
)

// engine_bench_test.go compares the JavaScript (goja) and Lua (gopher-lua +
// gopher-luar) engines across the operations the runtime actually performs:
// loading/compiling a script, looking up an entrypoint function, calling it
// with and without arguments, reflecting and mutating Go values, invoking a
// bound global Go function, and running a compute-heavy loop.
//
// Each benchmark uses sub-benchmarks named "js" and "lua" so the two engines
// can be compared directly, e.g.:
//
//	go test ./internal/scripting -bench BenchmarkEngine -benchmem
//
// The source pairs below are written to be semantically equivalent in both
// languages so the comparison reflects engine cost rather than differing work.

type engineCase struct {
	name   string
	lang   ScriptLang
	source string
}

// mustLoad builds a VM for a benchmark, failing the benchmark on error.
func mustLoad(b *testing.B, lang ScriptLang, source string) scriptVM {
	b.Helper()
	vm, err := loadVM("bench", ScriptSource{Lang: lang, Source: source}, nil)
	if err != nil {
		b.Fatalf("loadVM(%v) failed: %v", lang, err)
	}
	return vm
}

// BenchmarkEngineLoad measures the cost of compiling/initializing a fresh VM
// (including binding all engine globals) for each language.
func BenchmarkEngineLoad(b *testing.B) {
	cases := []engineCase{
		{name: "js", lang: LangJS, source: `function onCommand(rest) { return rest === "yes"; }`},
		{name: "lua", lang: LangLua, source: `function onCommand(rest) return rest == "yes" end`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				vm, err := loadVM("bench", ScriptSource{Lang: tc.lang, Source: tc.source}, nil)
				if err != nil {
					b.Fatalf("loadVM failed: %v", err)
				}
				_ = vm
			}
		})
	}
}

// BenchmarkEngineGetFunction measures entrypoint lookup on an already-loaded
// VM. Both engines cache the resolved function inside the wrapper.
func BenchmarkEngineGetFunction(b *testing.B) {
	cases := []engineCase{
		{name: "js", lang: LangJS, source: `function onCommand(rest) { return rest === "yes"; }`},
		{name: "lua", lang: LangLua, source: `function onCommand(rest) return rest == "yes" end`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vm := mustLoad(b, tc.lang, tc.source)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if _, ok := vm.GetFunction("onCommand"); !ok {
					b.Fatal("onCommand not found")
				}
			}
		})
	}
}

// BenchmarkEngineCallNoArgs measures the cost of a no-argument call returning a
// constant. This isolates per-call overhead.
func BenchmarkEngineCallNoArgs(b *testing.B) {
	cases := []engineCase{
		{name: "js", lang: LangJS, source: `function run() { return 42; }`},
		{name: "lua", lang: LangLua, source: `function run() return 42 end`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vm := mustLoad(b, tc.lang, tc.source)
			fn, ok := vm.GetFunction("run")
			if !ok {
				b.Fatal("run not found")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if _, err := vm.Call(time.Second, fn); err != nil {
					b.Fatalf("Call failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkEngineCallArgsBool measures a call that takes a string argument and
// returns a boolean, exercising argument marshaling and the return-value path
// the event dispatchers rely on.
func BenchmarkEngineCallArgsBool(b *testing.B) {
	cases := []engineCase{
		{name: "js", lang: LangJS, source: `function onCommand(rest) { return rest === "yes"; }`},
		{name: "lua", lang: LangLua, source: `function onCommand(rest) return rest == "yes" end`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vm := mustLoad(b, tc.lang, tc.source)
			fn, ok := vm.GetFunction("onCommand")
			if !ok {
				b.Fatal("onCommand not found")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				res, err := vm.Call(time.Second, fn, vm.ToValue("yes"))
				if err != nil {
					b.Fatalf("Call failed: %v", err)
				}
				if v, ok := res.Export().(bool); !ok || !v {
					b.Fatalf("expected true, got %#v", res.Export())
				}
			}
		})
	}
}

// BenchmarkEngineGoMethodCall measures reflecting and invoking a Go method on a
// wrapped Go value, including the pointer mutation both engines support. This
// reflects the dominant cost of real scripts, which call wrapper methods on
// actor/room/item objects.
func BenchmarkEngineGoMethodCall(b *testing.B) {
	cases := []engineCase{
		{name: "js", lang: LangJS, source: `function run(c) { c.Add(1); }`},
		{name: "lua", lang: LangLua, source: `function run(c) c:Add(1) end`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vm := mustLoad(b, tc.lang, tc.source)
			fn, ok := vm.GetFunction("run")
			if !ok {
				b.Fatal("run not found")
			}
			c := &counter{}
			arg := vm.ToValue(c)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if _, err := vm.Call(time.Second, fn, arg); err != nil {
					b.Fatalf("Call failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkEngineGlobalGoFunc measures calling a bound global Go function
// (RandInt) from script. Both engines bind it via setUtilFunctions.
func BenchmarkEngineGlobalGoFunc(b *testing.B) {
	cases := []engineCase{
		{name: "js", lang: LangJS, source: `function run() { return RandInt(1, 100); }`},
		{name: "lua", lang: LangLua, source: `function run() return RandInt(1, 100) end`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vm := mustLoad(b, tc.lang, tc.source)
			fn, ok := vm.GetFunction("run")
			if !ok {
				b.Fatal("run not found")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if _, err := vm.Call(time.Second, fn); err != nil {
					b.Fatalf("Call failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkEngineComputeLoop measures a compute-heavy in-script loop (summing
// 0..999). This isolates raw interpreter throughput from the call/marshal
// overhead measured by the other benchmarks.
func BenchmarkEngineComputeLoop(b *testing.B) {
	cases := []engineCase{
		{
			name:   "js",
			lang:   LangJS,
			source: `function run() { var s = 0; for (var i = 0; i < 1000; i++) { s += i; } return s; }`,
		},
		{
			name:   "lua",
			lang:   LangLua,
			source: `function run() local s = 0 for i = 0, 999 do s = s + i end return s end`,
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			vm := mustLoad(b, tc.lang, tc.source)
			fn, ok := vm.GetFunction("run")
			if !ok {
				b.Fatal("run not found")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if _, err := vm.Call(time.Second, fn); err != nil {
					b.Fatalf("Call failed: %v", err)
				}
			}
		})
	}
}
