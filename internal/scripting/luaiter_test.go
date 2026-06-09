package scripting

import (
	"testing"
	"time"

	luar "layeh.com/gopher-luar"
)

// TestLuaIteratesGoContainers verifies that pairs()/ipairs() transparently
// iterate gopher-luar userdata wrapping Go maps and slices, matching the
// JavaScript engine's for...in / for...of behaviour. This is the regression
// guard for the guard-hungry onIdle "table expected, got userdata" failure.
func TestLuaIteratesGoContainers(t *testing.T) {
	store := map[string]any{}
	set := func(k string, v any) { store[k] = v }
	get := func(k string) any { return store[k] }

	src := `
function put()
	local t = {}
	t["101"] = 6
	t["102"] = 7
	setData("d", t)
end

function countPairs()
	local v = getData("d")   -- comes back as luar userdata
	local n = 0
	for k, val in pairs(v) do n = n + 1 end
	return n
end

function sumSlice(s)
	local total = 0
	for i, v in ipairs(s) do total = total + v end
	return total
end`

	vm, err := loadVM("test", ScriptSource{Lang: LangLua, Source: src}, nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	L := vm.(*luaVM).L
	L.SetGlobal("setData", luar.New(L, set))
	L.SetGlobal("getData", luar.New(L, get))

	putFn, _ := vm.GetFunction("put")
	if _, err := vm.Call(time.Second, putFn); err != nil {
		t.Fatalf("put: %v", err)
	}

	countFn, _ := vm.GetFunction("countPairs")
	res, err := vm.Call(time.Second, countFn)
	if err != nil {
		t.Fatalf("countPairs: %v", err)
	}
	if n, _ := res.Export().(float64); n != 2 {
		t.Fatalf("expected pairs over Go map to yield 2 entries, got %v", res.Export())
	}

	sumFn, _ := vm.GetFunction("sumSlice")
	res, err = vm.Call(time.Second, sumFn, luar.New(L, []int{1, 2, 3, 4}))
	if err != nil {
		t.Fatalf("sumSlice: %v", err)
	}
	if n, _ := res.Export().(float64); n != 10 {
		t.Fatalf("expected ipairs over Go slice to sum to 10, got %v", res.Export())
	}
}
