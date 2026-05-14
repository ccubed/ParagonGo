package scripting

import (
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/dop251/goja"
)

func setPanelFunctions(vm *goja.Runtime) {
	vm.Set(`PanelLayoutLoad`, PanelLayoutLoad)
	vm.Set(`PanelLayoutNew`, PanelLayoutNew)
}

// PanelLayoutLoad loads a panel layout from the datafiles panel-layouts directory.
// name is relative to panel-layouts/ and has no extension.
// Example: PanelLayoutLoad("character/status")
// Returns a ScriptPanelLayout on success, or throws a JavaScript error on failure.
func PanelLayoutLoad(name string) *ScriptPanelLayout {
	layout, err := templates.LoadPanelLayout(name)
	if err != nil {
		panic(err.Error())
	}
	return &ScriptPanelLayout{layout: layout}
}

// PanelLayoutNew creates a panel layout entirely in script, without a YAML file.
// opts is an optional JS object with any of: border, charset, gap, margin.
//
// Example:
//
//	var layout = PanelLayoutNew({ border: "full", charset: "rounded", gap: 1, margin: 1 });
//	var slot = layout.AddSlot();
//	slot.AddRow(["myPanel"]);
//	layout.Panel("myPanel").SetTitle(' <ansi fg="20">Stats</ansi> ').SetMinWidth(30);
//	layout.Panel("myPanel").Add("Str:", "S:", "42");
//	SendText(layout.Render());
func PanelLayoutNew(opts ...map[string]any) *ScriptPanelLayout {
	border := "full"
	charset := "single"
	gap := 1
	margin := 0

	if len(opts) > 0 && opts[0] != nil {
		o := opts[0]
		if v, ok := o["border"].(string); ok {
			border = v
		}
		if v, ok := o["charset"].(string); ok {
			charset = v
		}
		if v, ok := o["gap"].(int); ok {
			gap = v
		} else if v, ok := o["gap"].(int64); ok {
			gap = int(v)
		}
		if v, ok := o["margin"].(int); ok {
			margin = v
		} else if v, ok := o["margin"].(int64); ok {
			margin = int(v)
		}
	}

	layout := templates.NewPanelLayout(border, charset, gap, margin)
	return &ScriptPanelLayout{layout: layout}
}

// ScriptPanelLayout is the script-facing wrapper for a PanelLayout.
type ScriptPanelLayout struct {
	layout *templates.PanelLayout
}

// AddSlot adds a new vertical slot (column) to the layout and returns a
// ScriptPanelSlot for adding rows of panels into it.
func (l *ScriptPanelLayout) AddSlot() *ScriptPanelSlot {
	slot := l.layout.AddSlot()
	return &ScriptPanelSlot{layout: l.layout, slot: slot}
}

// Panel returns the named panel for data population.
// Panics with a descriptive message if the id does not exist.
func (l *ScriptPanelLayout) Panel(id string) *ScriptPanel {
	return &ScriptPanel{panel: l.layout.Panel(id)}
}

// Render synthesizes the layout into a terminal string.
func (l *ScriptPanelLayout) Render() string {
	return l.layout.Render()
}

// ScriptPanelSlot is the script-facing wrapper for a layout slot.
type ScriptPanelSlot struct {
	layout *templates.PanelLayout
	slot   *templates.LayoutSlot
}

// AddRow adds a horizontal row of panels to this slot.
// ids is an array of panel IDs to create in this row.
// Each panel is created with default settings; use layout.Panel(id) to configure it.
// Returns the slot for chaining.
//
// Example:
//
//	slot.AddRow(["info", "stats"]);
func (s *ScriptPanelSlot) AddRow(ids []string) *ScriptPanelSlot {
	s.layout.AddPanelsToSlot(s.slot, ids...)
	return s
}

// ScriptPanel is the script-facing wrapper for a Panel.
type ScriptPanel struct {
	panel *templates.Panel
}

// SetLabelWidth sets a fixed visual width that all labels are padded to.
// When non-zero, values always start at the same column regardless of label length.
// Returns the panel for chaining.
func (p *ScriptPanel) SetLabelWidth(w int) *ScriptPanel {
	p.panel.SetLabelWidth(w)
	return p
}

// SetCharset overrides the border character set for this panel.
// Recognised values: "single", "double", "rounded".
// If not called, the panel uses the layout-level charset.
// Returns the panel for chaining.
func (p *ScriptPanel) SetCharset(name string) *ScriptPanel {
	p.panel.SetCharset(name)
	return p
}

// SetTitle sets the panel's title string (used verbatim in the top border).
// May contain ANSI tags; visual width is measured with tags stripped.
// Returns the panel for chaining.
func (p *ScriptPanel) SetTitle(title string) *ScriptPanel {
	p.panel.SetTitle(title)
	return p
}

// SetMinWidth sets the panel's minimum inner content width.
// Returns the panel for chaining.
func (p *ScriptPanel) SetMinWidth(w int) *ScriptPanel {
	p.panel.SetMinWidth(w)
	return p
}

// SetColumns sets the number of label+value pairs per line (1 or 2).
// Returns the panel for chaining.
func (p *ScriptPanel) SetColumns(n int) *ScriptPanel {
	p.panel.SetColumns(n)
	return p
}

// SetColumnGap sets the spaces between columns when columns > 1.
// Returns the panel for chaining.
func (p *ScriptPanel) SetColumnGap(n int) *ScriptPanel {
	p.panel.SetColumnGap(n)
	return p
}

// Add appends a label+value row to the panel.
// fullLabel is used when it fits; shortLabel is the fallback.
// Returns the panel for chaining.
func (p *ScriptPanel) Add(fullLabel, shortLabel, value string) *ScriptPanel {
	p.panel.Add(fullLabel, shortLabel, value)
	return p
}

// AddBlank appends an empty spacer row. Returns the panel for chaining.
func (p *ScriptPanel) AddBlank() *ScriptPanel {
	p.panel.AddBlank()
	return p
}
