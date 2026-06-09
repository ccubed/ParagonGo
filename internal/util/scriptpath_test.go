package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveScriptPath(t *testing.T) {
	dir := t.TempDir()

	yamlPath := filepath.Join(dir, "5-thing.yaml")
	jsPath := filepath.Join(dir, "5-thing.js")
	luaPath := filepath.Join(dir, "5-thing.lua")

	write := func(p string) {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	// Neither exists: defaults to the .js path.
	if got := ResolveScriptPath(yamlPath); got != jsPath {
		t.Fatalf("no script: expected %q, got %q", jsPath, got)
	}

	// Only lua exists: returns the .lua path.
	write(luaPath)
	if got := ResolveScriptPath(yamlPath); got != luaPath {
		t.Fatalf("lua only: expected %q, got %q", luaPath, got)
	}

	// Both exist: JavaScript wins.
	write(jsPath)
	if got := ResolveScriptPath(yamlPath); got != jsPath {
		t.Fatalf("both present: expected %q (js wins), got %q", jsPath, got)
	}

	// Only js exists.
	if err := os.Remove(luaPath); err != nil {
		t.Fatalf("remove lua: %v", err)
	}
	if got := ResolveScriptPath(yamlPath); got != jsPath {
		t.Fatalf("js only: expected %q, got %q", jsPath, got)
	}
}
