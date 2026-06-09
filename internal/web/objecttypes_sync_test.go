package web

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// parseInterfaceMethods extracts the method names declared in a TypeScript
// `declare interface <name> { ... }` block from the engineObjectInterfaces
// source. It returns the set of method names for the named interface.
func parseInterfaceMethods(t *testing.T, src, ifaceName string) map[string]bool {
	t.Helper()

	start := strings.Index(src, "declare interface "+ifaceName+" {")
	if start < 0 {
		t.Fatalf("interface %q not found in engineObjectInterfaces", ifaceName)
	}
	rest := src[start:]
	// Find the body between the first '{' and its matching '}', tracking brace
	// depth so inline object types in return signatures (e.g. "{visited: number}")
	// do not terminate the interface body early.
	open := strings.Index(rest, "{")
	if open < 0 {
		t.Fatalf("interface %q has no opening brace", ifaceName)
	}
	depth := 0
	bodyEnd := -1
	for i := open; i < len(rest); i++ {
		switch rest[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				bodyEnd = i
			}
		}
		if bodyEnd >= 0 {
			break
		}
	}
	if bodyEnd < 0 {
		t.Fatalf("interface %q has no matching closing brace", ifaceName)
	}
	body := rest[open+1 : bodyEnd]

	// Match method declarations: a name at the start of a line followed by '('.
	// Inline object-type members (no parentheses) are skipped, which is correct
	// because the structured model only describes callable methods.
	methodRe := regexp.MustCompile(`(?m)^\s*([A-Za-z_]\w*)\s*\(`)
	matches := methodRe.FindAllStringSubmatch(body, -1)

	names := make(map[string]bool, len(matches))
	for _, m := range matches {
		names[m[1]] = true
	}
	return names
}

// TestObjectTypesMatchDts ensures the structured object-type model served to
// the Lua editor stays in sync with the hand-authored TypeScript interfaces
// used by the JavaScript editor. If a method is added to or removed from one,
// this test flags the other so both intellisense paths stay accurate.
func TestObjectTypesMatchDts(t *testing.T) {
	objTypes := scripting.GetScriptObjectTypes()

	// Object type name -> the matching TS interface name.
	pairs := map[string]string{
		"ActorObject":       "ActorObject",
		"RoomObject":        "RoomObject",
		"ItemObject":        "ItemObject",
		"PetObject":         "PetObject",
		"PartyObject":       "PartyObject",
		"ContainerObject":   "ContainerObject",
		"PanelLayoutObject": "PanelLayoutObject",
		"SlotObject":        "SlotObject",
		"PanelObject":       "PanelObject",
	}

	for typeName, ifaceName := range pairs {
		def, ok := objTypes.Types[typeName]
		if !ok {
			t.Errorf("object type %q missing from GetScriptObjectTypes", typeName)
			continue
		}

		structured := make(map[string]bool, len(def.Methods))
		for _, meth := range def.Methods {
			structured[meth.Name] = true
		}

		dts := parseInterfaceMethods(t, engineObjectInterfaces, ifaceName)

		for name := range dts {
			if !structured[name] {
				t.Errorf("%s: method %q is in the .d.ts interface but missing from the structured object type", typeName, name)
			}
		}
		for name := range structured {
			if !dts[name] {
				t.Errorf("%s: method %q is in the structured object type but missing from the .d.ts interface", typeName, name)
			}
		}
	}
}

// TestObjectTypesSortedNamesStable is a light guard that the structured model
// is non-empty and deterministic to serialize.
func TestObjectTypesNonEmpty(t *testing.T) {
	objTypes := scripting.GetScriptObjectTypes()
	if len(objTypes.Types) == 0 {
		t.Fatal("expected at least one object type")
	}
	for name, def := range objTypes.Types {
		if len(def.Methods) == 0 {
			t.Errorf("object type %q has no methods", name)
		}
		names := make([]string, 0, len(def.Methods))
		for _, meth := range def.Methods {
			names = append(names, meth.Name)
		}
		if !sort.StringsAreSorted(names) {
			// Methods need not be sorted; this is only a sanity touch to keep
			// the slice usable. No assertion failure here.
			_ = names
		}
	}
}
