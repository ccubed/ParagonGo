package gmcp

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// GMCPBiomeColor holds pre-converted CSS hex colors for a biome or symbol
// override. An empty string means no color override is set.
type GMCPBiomeColor struct {
	FG string `json:"fg"` // e.g. "#00aa00", "" = no override
	BG string `json:"bg"` // e.g. "#003300", "" = no override
}

// GMCPBiomeEntry is the per-biome record included in the World.Map biomes
// lookup table. It contains everything window-map.js needs to colorize rooms
// without making any additional API calls.
type GMCPBiomeEntry struct {
	Name            string                    `json:"name"`
	Symbol          string                    `json:"symbol"`
	Color           GMCPBiomeColor            `json:"color"`
	SymbolOverrides map[string]GMCPBiomeColor `json:"overrides,omitempty"`
}

// ansi256ToHex converts an ANSI 256-color index to a CSS hex string.
// Returns "" for index 0 (treated as "no override") or out-of-range values.
// The conversion mirrors the JS ansi256ToHex in mapper-helpers.js exactly.
func ansi256ToHex(n int) string {
	if n <= 0 || n > 255 {
		return ""
	}

	// Indices 1-15: named 16-color palette (index 0 = no override, skipped above)
	named := []string{
		"#000000", "#800000", "#008000", "#808000",
		"#000080", "#800080", "#008080", "#c0c0c0",
		"#808080", "#ff0000", "#00ff00", "#ffff00",
		"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
	}
	if n < 16 {
		return named[n]
	}

	// Indices 16-231: 6x6x6 color cube
	if n < 232 {
		idx := n - 16
		bv := idx % 6
		gv := (idx / 6) % 6
		rv := idx / 36
		cv := func(i int) int {
			if i == 0 {
				return 0
			}
			return 55 + i*40
		}
		return fmt.Sprintf("#%02x%02x%02x", cv(rv), cv(gv), cv(bv))
	}

	// Indices 232-255: grayscale ramp
	gray := 8 + (n-232)*10
	return fmt.Sprintf("#%02x%02x%02x", gray, gray, gray)
}

// buildBiomeTable returns a map keyed by biomeId containing the color and
// symbol data needed by the client map renderer. Called once per World.Map
// payload build.
func buildBiomeTable() map[string]GMCPBiomeEntry {
	allBiomes := rooms.GetAllBiomes()
	table := make(map[string]GMCPBiomeEntry, len(allBiomes))

	for _, b := range allBiomes {
		entry := GMCPBiomeEntry{
			Name:   b.Name,
			Symbol: b.Symbol,
			Color: GMCPBiomeColor{
				FG: ansi256ToHex(b.Color.FGColor),
				BG: ansi256ToHex(b.Color.BGColor),
			},
		}

		if len(b.SymbolOverrides) > 0 {
			entry.SymbolOverrides = make(map[string]GMCPBiomeColor, len(b.SymbolOverrides))
			for sym, sc := range b.SymbolOverrides {
				entry.SymbolOverrides[sym] = GMCPBiomeColor{
					FG: ansi256ToHex(sc.FGColor),
					BG: ansi256ToHex(sc.BGColor),
				}
			}
		}

		table[b.BiomeId] = entry
	}

	return table
}
