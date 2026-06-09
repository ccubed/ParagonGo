package scripting

import (
	"testing"
	"time"

	luar "layeh.com/gopher-luar"
)

// fakeIdlePlayer is a stand-in for a player actor with the two methods the
// onIdle path calls.
type fakeIdlePlayer struct {
	id        int
	hasQuest  bool
	questName string
}

func (p *fakeIdlePlayer) UserId() int { return p.id }
func (p *fakeIdlePlayer) HasQuest(q string) bool {
	return p.hasQuest && p.questName == q
}

// TestGuardHungryIdleTempData runs the exact onIdle playersTold logic against a
// stub temp-data store to confirm the read/copy/mutate/persist cycle works with
// no "table expected, got userdata" errors across repeated rounds.
func TestGuardHungryIdleTempData(t *testing.T) {
	tempStore := map[string]any{}

	// onIdleStep mirrors the script's onIdle dictionary handling, taking the
	// players slice and round explicitly so the test can drive multiple rounds.
	src := `
function onIdleStep(players, round, getData, setData)
	local playersTold = {}
	local stored = getData("playersTold")
	if stored ~= nil then
		for k, v in pairs(stored) do
			playersTold[k] = v
		end
	end

	if #players > 0 then
		for i = 1, #players do
			local uid = tostring(players[i]:UserId())
			if playersTold[uid] ~= nil then
				if round < playersTold[uid] then
					goto continue
				end
			end
			if not players[i]:HasQuest("4-start") then
				playersTold[uid] = round + 5
			else
				playersTold[uid] = round + 500
			end
			do break end
			::continue::
		end

		if next(playersTold) ~= nil then
			setData("playersTold", playersTold)
		else
			setData("playersTold", nil)
		end
		return "active"
	end

	for key, value in pairs(playersTold) do
		if value < round - 100 then
			playersTold[key] = nil
		end
	end
	if next(playersTold) == nil then
		setData("playersTold", nil)
	else
		setData("playersTold", playersTold)
	end
	return "idle"
end`

	vm, err := loadVM("test", ScriptSource{Lang: LangLua, Source: src}, nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	L := vm.(*luaVM).L

	get := luar.New(L, func(k string) any { return tempStore[k] })
	set := luar.New(L, func(k string, v any) {
		if v == nil {
			delete(tempStore, k)
			return
		}
		tempStore[k] = v
	})

	players := []*fakeIdlePlayer{{id: 101}, {id: 102}}

	step, _ := vm.GetFunction("onIdleStep")

	// Several rounds with players present: must never error, and must persist
	// playersTold across calls (proving the userdata round-trip works).
	for round := 1; round <= 3; round++ {
		res, err := vm.Call(time.Second, step, luar.New(L, players), round, get, set)
		if err != nil {
			t.Fatalf("round %d errored (userdata bug regression?): %v", round, err)
		}
		if s, _ := res.Export().(string); s != "active" {
			t.Fatalf("round %d expected active, got %q", round, s)
		}
		if _, ok := tempStore["playersTold"]; !ok {
			t.Fatalf("round %d expected playersTold to be persisted", round)
		}
	}

	// No players present and round far in the future: cleanup empties the map
	// and removes the temp data without error.
	res, err := vm.Call(time.Second, step, luar.New(L, []*fakeIdlePlayer{}), 100000, get, set)
	if err != nil {
		t.Fatalf("idle cleanup errored: %v", err)
	}
	if s, _ := res.Export().(string); s != "idle" {
		t.Fatalf("expected idle, got %q", s)
	}
	if _, ok := tempStore["playersTold"]; ok {
		t.Fatalf("expected playersTold cleared after cleanup")
	}
}
