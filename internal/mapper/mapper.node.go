package mapper

import "github.com/GoMudEngine/GoMud/internal/rooms"

// represents a single room
type mapNode struct {
	RoomId          int
	Symbol          rune
	Legend          string                       // The same that shows in the legend for this symbol
	Color           rooms.SymbolColor            // FG/BG color from biome (zero values = use alias fallback)
	SymbolOverrides map[string]rooms.SymbolColor // Per-symbol color overrides from the biome
	Exits           map[string]nodeExit
	SecretExits     map[string]struct{} // Just a flag for whether an exit key is secret
	Pos             positionDelta       // Its x/y/z position relative to the root node
	HasStoredCoords bool
}

type nodeExit struct {
	RoomId         int    // where it leads to
	Secret         bool   // is it secret?
	LockDifficulty int    // If > 0, the lock difficulty.
	LockId         string // What's the lock id?
	Direction      positionDelta
}
