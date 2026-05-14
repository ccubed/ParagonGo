package templates

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/ansitags"
	"github.com/mattn/go-runewidth"
	"gopkg.in/yaml.v2"
)

type borderStyle string

const (
	borderFull borderStyle = "full"
	borderOpen borderStyle = "open"

	panelPad         = 1 // spaces inside border on each side of content
	defaultColumnGap = 2 // spaces between columns when columns > 1
)

// borderChars holds the six characters used to draw a panel border.
type borderChars struct {
	TopLeft     string // e.g. ┌
	TopRight    string // e.g. ┐
	BottomLeft  string // e.g. └
	BottomRight string // e.g. ┘
	Horizontal  string // e.g. ─
	Vertical    string // e.g. │
}

var (
	charsetSingle  = borderChars{"┌", "┐", "└", "┘", "─", "│"}
	charsetDouble  = borderChars{"╔", "╗", "╚", "╝", "═", "║"}
	charsetRounded = borderChars{"╭", "╮", "╰", "╯", "─", "│"}
)

func charsetForName(name string) borderChars {
	switch strings.ToLower(name) {
	case "double":
		return charsetDouble
	case "rounded":
		return charsetRounded
	default:
		return charsetSingle
	}
}

// PanelRow is one label+value line inside a panel.
// The renderer uses FullLabel when it fits the panel width, ShortLabel otherwise.
// Set Blank to true to insert an empty spacer line; label and value are ignored.
type PanelRow struct {
	FullLabel  string
	ShortLabel string
	Value      string
	Blank      bool
}

// Panel holds the rows for one titled box. Obtain via PanelLayout.Panel(id).
type Panel struct {
	id         string
	title      string // raw title string, may contain ANSI tags; used verbatim in the top border
	minWidth   int
	border     borderStyle
	chars      borderChars
	columns    int // 1 (default) or 2: how many label+value pairs share a line
	columnGap  int // spaces between columns when columns > 1
	labelWidth int // if > 0, all labels are right-padded to this visual width
	rows       []PanelRow
}

// Add appends a label+value row and returns the panel for chaining.
func (p *Panel) Add(fullLabel, shortLabel, value string) *Panel {
	p.rows = append(p.rows, PanelRow{
		FullLabel:  fullLabel,
		ShortLabel: shortLabel,
		Value:      value,
	})
	return p
}

// AddBlank appends an empty spacer row and returns the panel for chaining.
func (p *Panel) AddBlank() *Panel {
	p.rows = append(p.rows, PanelRow{Blank: true})
	return p
}

// layoutSlot is one vertical column in the layout.
// It contains one or more horizontal rows of panels stacked top-to-bottom.
type layoutSlot struct {
	rows [][]*Panel
}

// renderSlot renders a slot's stacked rows into a single []string column.
// All lines are padded to the same visual width (the widest row in the slot)
// so that the slot can be composed side-by-side with other slots cleanly.
func renderSlot(slot *layoutSlot, gap int) []string {
	var result []string
	for _, row := range slot.rows {
		result = append(result, composePanelGroup(row, gap)...)
	}
	if len(result) == 0 {
		return result
	}
	// Find the widest line in this slot.
	maxW := 0
	for _, line := range result {
		if w := panelVisualWidth(line); w > maxW {
			maxW = w
		}
	}
	// Right-pad any shorter lines so all lines have the same visual width.
	for i, line := range result {
		if w := panelVisualWidth(line); w < maxW {
			result[i] = line + strings.Repeat(" ", maxW-w)
		}
	}
	return result
}

// PanelLayout is a loaded layout skeleton populated with data and ready to render.
type PanelLayout struct {
	border borderStyle
	chars  borderChars
	gap    int
	margin int               // spaces prepended to every output line
	slots  []*layoutSlot     // top-level horizontal slots (columns)
	byID   map[string]*Panel // fast lookup by panel id
}

// Panel returns the named panel for data population.
// It panics with a descriptive message if the id is not defined in the layout.
func (l *PanelLayout) Panel(id string) *Panel {
	p, ok := l.byID[id]
	if !ok {
		panic(fmt.Sprintf("panel layout: no panel with id %q", id))
	}
	return p
}

// Render synthesizes all slots into a single terminal string.
func (l *PanelLayout) Render() string {
	if len(l.slots) == 0 {
		return ""
	}

	// Render each slot to its own []string column.
	rendered := make([][]string, len(l.slots))
	for i, slot := range l.slots {
		rendered[i] = renderSlot(slot, l.gap)
	}

	// Find the maximum line count across all slots.
	maxLines := 0
	for _, col := range rendered {
		if len(col) > maxLines {
			maxLines = len(col)
		}
	}

	// Pad shorter slots with blank lines of the correct visual width.
	for i, col := range rendered {
		if len(col) >= maxLines {
			continue
		}
		// Measure the visual width of this slot from its first line.
		w := 0
		if len(col) > 0 {
			w = panelVisualWidth(col[0])
		}
		blank := strings.Repeat(" ", w)
		for len(rendered[i]) < maxLines {
			rendered[i] = append(rendered[i], blank)
		}
	}

	gapStr := strings.Repeat(" ", l.gap)
	out := make([]string, maxLines)
	for row := 0; row < maxLines; row++ {
		parts := make([]string, len(rendered))
		for i, col := range rendered {
			parts[i] = col[row]
		}
		out[row] = strings.Join(parts, gapStr)
	}
	if l.margin > 0 {
		prefix := strings.Repeat(" ", l.margin)
		for i, line := range out {
			out[i] = prefix + line
		}
	}
	return strings.Join(out, "\n")
}

// ---------------------------------------------------------------------------
// YAML definition structs
// ---------------------------------------------------------------------------

// panelDef is the YAML structure for a single panel entry.
type panelDef struct {
	ID        string `yaml:"id"`
	Title     string `yaml:"title"`
	MinWidth  int    `yaml:"min_width"`
	Columns   int    `yaml:"columns"`    // optional, default 1
	ColumnGap int    `yaml:"column_gap"` // optional, default 2
	Charset   string `yaml:"charset"`    // optional, overrides layout-level charset
}

// panelRowDef is a horizontal group of panels in the YAML.
type panelRowDef struct {
	Panels []panelDef `yaml:"panels"`
}

// slotDef is one vertical slot (column) in the YAML.
// It contains one or more rows of panels stacked top-to-bottom.
type slotDef struct {
	Rows []panelRowDef `yaml:"rows"`
}

// panelLayoutDef is the top-level YAML structure.
type panelLayoutDef struct {
	Border  string    `yaml:"border"`
	Gap     int       `yaml:"gap"`
	Margin  int       `yaml:"margin"`  // optional left margin applied to every output line
	Charset string    `yaml:"charset"` // optional: "single" (default), "double", "rounded"
	Slots   []slotDef `yaml:"slots"`
}

// LayoutSlot is the exported handle for a slot, used by the scripting layer.
type LayoutSlot = layoutSlot

// NewPanelLayout creates a PanelLayout programmatically without loading a YAML file.
func NewPanelLayout(border, charset string, gap, margin int) *PanelLayout {
	b := borderFull
	if borderStyle(border) == borderOpen {
		b = borderOpen
	}
	if gap < 0 {
		gap = 0
	}
	return &PanelLayout{
		border: b,
		gap:    gap,
		margin: margin,
		chars:  charsetForName(charset),
		byID:   make(map[string]*Panel),
	}
}

// AddSlot appends a new empty slot to the layout and returns it.
func (l *PanelLayout) AddSlot() *LayoutSlot {
	slot := &layoutSlot{}
	l.slots = append(l.slots, slot)
	return slot
}

// AddPanelsToSlot appends a new row of panels (one per id) to the given slot.
// Panels are created with default settings. Use Panel(id) to configure them further.
func (l *PanelLayout) AddPanelsToSlot(slot *LayoutSlot, ids ...string) {
	var row []*Panel
	for _, id := range ids {
		p := &Panel{
			id:        id,
			border:    l.border,
			chars:     l.chars,
			columns:   1,
			columnGap: defaultColumnGap,
		}
		l.byID[id] = p
		row = append(row, p)
	}
	slot.rows = append(slot.rows, row)
}

// SetCharset sets the border character set for this panel, overriding the
// layout-level charset. Recognised values: "single", "double", "rounded".
// An unrecognised value falls back to "single".
func (p *Panel) SetCharset(name string) *Panel { p.chars = charsetForName(name); return p }

// SetTitle sets the panel's title string verbatim.
func (p *Panel) SetTitle(title string) *Panel { p.title = title; return p }

// SetMinWidth sets the panel's minimum inner content width.
func (p *Panel) SetMinWidth(w int) *Panel { p.minWidth = w; return p }

// SetLabelWidth sets a fixed visual width that all labels are padded to.
// When non-zero, every label is right-padded with spaces to this width before
// rendering, so values always start at the same column regardless of label length.
// ANSI tags in labels are accounted for correctly.
func (p *Panel) SetLabelWidth(w int) *Panel { p.labelWidth = w; return p }

// SetColumns sets the number of label+value pairs per rendered line (1 or 2).
func (p *Panel) SetColumns(n int) *Panel {
	if n < 1 {
		n = 1
	}
	p.columns = n
	return p
}

// SetColumnGap sets the spaces between columns when columns > 1.
func (p *Panel) SetColumnGap(n int) *Panel {
	if n < 0 {
		n = 0
	}
	p.columnGap = n
	return p
}

// LoadPanelLayout loads a panel layout definition from the datafiles directory.
// name is relative to the panel-layouts/ subdirectory and has no extension.
// Example: LoadPanelLayout("character/status")
func LoadPanelLayout(name string) (*PanelLayout, error) {
	dataFiles := string(configs.GetFilePathsConfig().DataFiles)
	path := dataFiles + "/panel-layouts/" + name + ".yaml"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("panel layout %q: %w", name, err)
	}

	var def panelLayoutDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("panel layout %q: %w", name, err)
	}

	border := borderFull
	if borderStyle(def.Border) == borderOpen {
		border = borderOpen
	}

	chars := charsetForName(def.Charset)

	gap := def.Gap
	if gap < 0 {
		gap = 0
	}

	layout := &PanelLayout{
		border: border,
		chars:  chars,
		gap:    gap,
		margin: def.Margin,
		byID:   make(map[string]*Panel),
	}

	for _, sd := range def.Slots {
		slot := &layoutSlot{}
		for _, rd := range sd.Rows {
			var rowPanels []*Panel
			for _, pd := range rd.Panels {
				cols := pd.Columns
				if cols < 1 {
					cols = 1
				}
				colGap := pd.ColumnGap
				if colGap < 1 {
					colGap = defaultColumnGap
				}
				panelChars := chars
				if pd.Charset != "" {
					panelChars = charsetForName(pd.Charset)
				}
				p := &Panel{
					id:        pd.ID,
					title:     pd.Title,
					minWidth:  pd.MinWidth,
					border:    border,
					chars:     panelChars,
					columns:   cols,
					columnGap: colGap,
				}
				layout.byID[pd.ID] = p
				rowPanels = append(rowPanels, p)
			}
			slot.rows = append(slot.rows, rowPanels)
		}
		layout.slots = append(layout.slots, slot)
	}

	return layout, nil
}

// ---------------------------------------------------------------------------
// Width helpers
// ---------------------------------------------------------------------------

// panelVisualWidth returns the visible terminal width of s, stripping ANSI tags.
func panelVisualWidth(s string) int {
	return runewidth.StringWidth(ansitags.Parse(s, ansitags.StripTags))
}

// panelInnerWidth calculates the inner content width for a panel.
// For single-column panels it is the max of minWidth and the widest row.
// For multi-column panels it is the max of minWidth and twice the widest
// half-column (each half = label+1+value), plus the column gap.
func panelInnerWidth(p *Panel) int {
	width := p.minWidth
	if p.columns < 2 {
		for _, row := range p.rows {
			if row.Blank {
				continue
			}
			lw := panelVisualWidth(row.FullLabel)
			if p.labelWidth > lw {
				lw = p.labelWidth
			}
			vw := panelVisualWidth(row.Value)
			needed := lw + 1 + vw
			if needed > width {
				width = needed
			}
		}
		return width
	}
	// Multi-column: find the widest single cell, then total = 2*colWidth + gap.
	widestCell := 0
	for _, row := range p.rows {
		if row.Blank {
			continue
		}
		lw := panelVisualWidth(row.FullLabel)
		vw := panelVisualWidth(row.Value)
		cell := lw + 1 + vw
		if cell > widestCell {
			widestCell = cell
		}
	}
	colWidth := (p.minWidth - p.columnGap) / 2
	if widestCell > colWidth {
		colWidth = widestCell
	}
	total := colWidth*2 + p.columnGap
	if total > width {
		width = total
	}
	return width
}

// chooseLabel returns the label to use for a row given the available inner width.
// It prefers FullLabel; falls back to ShortLabel if FullLabel doesn't fit alongside the value.
func chooseLabel(row PanelRow, innerWidth int) string {
	flw := panelVisualWidth(row.FullLabel)
	vw := panelVisualWidth(row.Value)
	if flw+1+vw <= innerWidth {
		return row.FullLabel
	}
	return row.ShortLabel
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// renderCellContent renders one label+value cell right-padded to colWidth.
func renderCellContent(row PanelRow, colWidth int) string {
	label := chooseLabel(row, colWidth)
	lw := panelVisualWidth(label)
	vw := panelVisualWidth(row.Value)
	rightPad := colWidth - lw - 1 - vw
	if rightPad < 0 {
		rightPad = 0
	}
	return label + " " + row.Value + strings.Repeat(" ", rightPad)
}

// renderPanel renders a single panel into a slice of terminal lines.
// Each line is a complete row including border characters and padding.
// Lines do not end with a newline.
func renderPanel(p *Panel) []string {
	inner := panelInnerWidth(p)
	c := p.chars

	var lines []string

	// Top border: TopLeft + Horizontal + " " + title + " " + Horizontal... + TopRight
	// The title is used verbatim (may contain ANSI tags); its visual width is measured stripped.
	titleVW := panelVisualWidth(p.title)
	// visible structure: TL + H + " " + title + " " + H*n + TR
	// that is 1 + 1 + 1 + titleVW + 1 + n + 1 = inner + 2*panelPad + 2
	dashCount := inner + 2*panelPad + 2 - 1 - 1 - 1 - titleVW - 1 - 1
	if dashCount < 0 {
		dashCount = 0
	}
	lines = append(lines, c.TopLeft+c.Horizontal+" "+p.title+" "+strings.Repeat(c.Horizontal, dashCount)+c.TopRight)

	nRows := len(p.rows)

	if p.columns < 2 {
		// Single-column layout.
		for i, row := range p.rows {
			isFirst := i == 0
			isLast := i == nRows-1
			lines = append(lines, renderSingleColumnLine(p, row, inner, isFirst, isLast))
		}
	} else {
		// Multi-column layout: pair up rows.
		colWidth := (inner - p.columnGap) / 2
		gapStr := strings.Repeat(" ", p.columnGap)

		for i := 0; i < nRows; i += p.columns {
			isFirst := i == 0
			isLast := i+p.columns >= nRows

			rowA := p.rows[i]
			hasB := i+1 < nRows

			var content string
			if rowA.Blank {
				content = strings.Repeat(" ", inner+2*panelPad)
			} else if !hasB || p.rows[i+1].Blank {
				// Odd row out or right cell is blank: span full width.
				content = strings.Repeat(" ", panelPad) +
					renderCellContent(rowA, inner) +
					strings.Repeat(" ", panelPad)
			} else {
				rowB := p.rows[i+1]
				cellA := renderCellContent(rowA, colWidth)
				cellB := renderCellContent(rowB, colWidth)
				content = strings.Repeat(" ", panelPad) +
					cellA + gapStr + cellB +
					strings.Repeat(" ", panelPad)
			}

			if p.border == borderFull || isFirst || isLast {
				lines = append(lines, c.Vertical+content+c.Vertical)
			} else {
				lines = append(lines, " "+content+" ")
			}
		}
	}

	// Bottom border
	lines = append(lines, c.BottomLeft+strings.Repeat(c.Horizontal, inner+2*panelPad)+c.BottomRight)

	return lines
}

// renderSingleColumnLine renders one content line for a single-column panel.
func renderSingleColumnLine(p *Panel, row PanelRow, inner int, isFirst, isLast bool) string {
	c := p.chars
	var content string
	if row.Blank {
		content = strings.Repeat(" ", inner+2*panelPad)
	} else {
		label := chooseLabel(row, inner)
		lw := panelVisualWidth(label)
		if p.labelWidth > lw {
			label = label + strings.Repeat(" ", p.labelWidth-lw)
			lw = p.labelWidth
		}
		vw := panelVisualWidth(row.Value)
		rightPad := inner - lw - 1 - vw
		if rightPad < 0 {
			rightPad = 0
		}
		content = strings.Repeat(" ", panelPad) +
			label + " " + row.Value +
			strings.Repeat(" ", rightPad) +
			strings.Repeat(" ", panelPad)
	}
	if p.border == borderFull || isFirst || isLast {
		return c.Vertical + content + c.Vertical
	}
	return " " + content + " "
}

// composePanelGroup renders a group of panels side by side into a slice of lines.
// Panels of different heights are padded so all have the same number of lines.
// The bottom border is always the last line; blank padding is inserted above it.
func composePanelGroup(panels []*Panel, gap int) []string {
	if len(panels) == 0 {
		return nil
	}

	rendered := make([][]string, len(panels))
	for i, p := range panels {
		rendered[i] = renderPanel(p)
	}

	// Find the maximum line count.
	maxLines := 0
	for _, lines := range rendered {
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}

	// Pad each panel's lines to maxLines.
	// The last line is always the bottom border; insert blank filler lines before it.
	for i, lines := range rendered {
		if len(lines) >= maxLines {
			continue
		}
		p := panels[i]
		inner := panelInnerWidth(p)
		blankContent := strings.Repeat(" ", inner+2*panelPad)

		var blankLine string
		if p.border == borderFull {
			blankLine = p.chars.Vertical + blankContent + p.chars.Vertical
		} else {
			blankLine = " " + blankContent + " "
		}

		needed := maxLines - len(lines)
		bottom := lines[len(lines)-1]
		middle := lines[:len(lines)-1]
		padding := make([]string, needed)
		for j := range padding {
			padding[j] = blankLine
		}
		rendered[i] = append(middle, append(padding, bottom)...)
	}

	gapStr := strings.Repeat(" ", gap)
	result := make([]string, maxLines)
	for row := 0; row < maxLines; row++ {
		var parts []string
		for _, lines := range rendered {
			parts = append(parts, lines[row])
		}
		result[row] = strings.Join(parts, gapStr)
	}
	return result
}
