package mapper

// MapCell holds one rendered character and its resolved ANSI colors.
type MapCell struct {
	Symbol  rune
	FGColor int // 0 = use map-{legend} alias fallback
	BGColor int // 0 = no background override
}

type mapRender struct {
	Render [][]MapCell
	legend map[rune]string // Symbol=>Name (for legend display only)
}

func newMapRender(mapWidth int, mapHeight int) mapRender {
	ret := mapRender{
		Render: make([][]MapCell, mapHeight),
		legend: make(map[rune]string, 3),
	}
	for y := 0; y < mapHeight; y++ {
		ret.Render[y] = make([]MapCell, mapWidth)
		for x := 0; x < mapWidth; x++ {
			ret.Render[y][x] = MapCell{Symbol: ' '}
		}
	}
	return ret
}

func (m *mapRender) GetLegend(overrides map[rune]string) map[rune]string {
	ret := map[rune]string{}
	for r, name := range m.legend {
		if overrides != nil {
			if oName, ok := overrides[r]; ok {
				ret[r] = oName
			} else {
				ret[r] = name
			}
		} else {
			ret[r] = name
		}
	}
	return ret
}
