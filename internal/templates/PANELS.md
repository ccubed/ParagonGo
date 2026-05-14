# Panel Layout — Go Reference

`internal/templates/panel.go`

The panel layout system produces richly formatted terminal displays made up of
titled boxes arranged in columns and stacked rows. It handles all border
drawing, ANSI-aware width measurement, label fallback, and multi-column row
pairing automatically.

---

## Table of contents

- [Concepts](#concepts)
- [Quick start](#quick-start)
- [YAML layout files](#yaml-layout-files)
  - [Top-level fields](#top-level-fields)
  - [Slot, row, and panel fields](#slot-row-and-panel-fields)
  - [Full example](#full-example)
- [API reference](#api-reference)
  - [LoadPanelLayout](#loadpanellayout)
  - [NewPanelLayout](#newpanellayout)
  - [PanelLayout.AddSlot](#panellayoutaddslot)
  - [PanelLayout.AddPanelsToSlot](#panellayoutaddpanelstoslot)
  - [PanelLayout.Panel](#panellayoutpanel)
  - [PanelLayout.Render](#panellayoutrender)
  - [Panel.Add](#paneladd)
  - [Panel.AddBlank](#paneladdblank)
  - [Panel.SetTitle](#panelsettitle)
  - [Panel.SetMinWidth](#panelsetminwidth)
  - [Panel.SetColumns](#panelsetcolumns)
  - [Panel.SetColumnGap](#panelsetcolumngap)
- [Width and ANSI tags](#width-and-ansi-tags)
- [Border styles](#border-styles)
- [Character sets](#character-sets)
- [Multi-column panels](#multi-column-panels)
- [Margin](#margin)

---

## Concepts

A layout is composed of three nested levels:

```
PanelLayout
 ├── LayoutSlot  (vertical column)
 │    ├── []*Panel  (row 0 — panels placed side by side)
 │    └── []*Panel  (row 1 — stacked below row 0)
 └── LayoutSlot
      └── []*Panel  (row 0)
```

**Slots** are the top-level vertical columns. They are rendered side by side.
**Rows** live inside a slot and stack vertically. Each row holds one or more
**panels** placed side by side.

The simplest useful layout is one slot with one row containing one panel.

---

## Quick start

### Using a YAML file

Define the structure in a file under
`_datafiles/<world>/panel-layouts/<name>.yaml`, then load and populate it:

```go
layout, err := templates.LoadPanelLayout("character/status")
if err != nil {
    // fall back or return error
}

layout.Panel("info").
    Add(`<ansi fg="yellow">Name:  </ansi>`, `<ansi fg="yellow">N:</ansi>`, character.Name).
    Add(`<ansi fg="yellow">Level: </ansi>`, `<ansi fg="yellow">Lvl:</ansi>`, fmt.Sprintf("%d", character.Level)).
    Add(`<ansi fg="yellow">Health:</ansi>`, `<ansi fg="yellow">HP:</ansi>`, fmt.Sprintf("%d/%d", character.Health, character.HealthMax.Value))

output := layout.Render()
user.SendText(output)
```

### Building entirely in Go

```go
layout := templates.NewPanelLayout("full", "single", 1, 1)

slot := layout.AddSlot()
layout.AddPanelsToSlot(slot, "info", "stats")

layout.Panel("info").
    SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Info</ansi> `).
    SetMinWidth(28).
    Add(`<ansi fg="yellow">Name:</ansi>`, `<ansi fg="yellow">N:</ansi>`, character.Name)

layout.Panel("stats").
    SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Stats</ansi> `).
    SetMinWidth(30).
    SetColumns(2).
    Add(`<ansi fg="yellow">Strength:</ansi>`, `<ansi fg="yellow">Str:</ansi>`, fmt.Sprintf("%d", str)).
    Add(`<ansi fg="yellow">Vitality:</ansi>`, `<ansi fg="yellow">Vit:</ansi>`, fmt.Sprintf("%d", vit))

output := layout.Render()
```

---

## YAML layout files

Layout files live at:

```
_datafiles/<world>/panel-layouts/<path>.yaml
```

`LoadPanelLayout("character/status")` reads from
`<datafiles>/panel-layouts/character/status.yaml`.

### Top-level fields

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `border` | string | `"full"` | `"full"` or `"open"` — see [Border styles](#border-styles) |
| `charset` | string | `"single"` | `"single"`, `"double"`, or `"rounded"` — see [Character sets](#character-sets) |
| `gap` | int | `0` | Spaces between panels in the same row and between slots |
| `margin` | int | `0` | Spaces prepended to every output line |
| `slots` | list | — | Ordered list of slot definitions |

### Slot, row, and panel fields

Each slot contains a list of rows. Each row contains a list of panels.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `slots[].rows` | list | — | Ordered list of row definitions in this slot |
| `slots[].rows[].panels` | list | — | Ordered list of panel definitions in this row |
| `panels[].id` | string | — | **Required.** Identifier used in `layout.Panel(id)` |
| `panels[].title` | string | `""` | Title string rendered verbatim into the top border. May contain ANSI tags. |
| `panels[].min_width` | int | `0` | Minimum inner content width; expands if content is wider |
| `panels[].columns` | int | `1` | `1` or `2` — see [Multi-column panels](#multi-column-panels) |
| `panels[].column_gap` | int | `2` | Spaces between columns when `columns: 2` |
| `panels[].charset` | string | _(layout charset)_ | Optional per-panel charset override: `"single"`, `"double"`, or `"rounded"` |

### Full example

```yaml
# border: "full"    - every row has side borders  │ label  value │
# border: "open"    - only first/last rows have side borders
#
# charset: "single"  - ┌─┐ │ └─┘
# charset: "double"  - ╔═╗ ║ ╚═╝
# charset: "rounded" - ╭─╮ │ ╰─╯

border: open
charset: single
gap: 1
margin: 1

slots:
  - rows:
      - panels:
          - id: info
            title: ' <ansi fg="black-bold">.:</ansi><ansi fg="20">Info</ansi> '
            min_width: 30

  - rows:
      - panels:
          - id: attributes
            title: ' <ansi fg="black-bold">.:</ansi><ansi fg="20">Attributes</ansi> '
            min_width: 42
            columns: 2
            column_gap: 2
      - panels:
          - id: wealth
            title: ' <ansi fg="black-bold">.:</ansi><ansi fg="20">Wealth</ansi> '
            min_width: 19
          - id: training
            title: ' <ansi fg="black-bold">.:</ansi><ansi fg="20">Training</ansi> '
            min_width: 20
```

This produces:

```
 ┌─ .:Info ─────────────────────┐ ┌─ .:Attributes ──────────────────────────┐
 │ Area:   Frostfang             │ │ Strength:  42(+3)  Vitality:  38(+0)    │
   Race:   Human (medium)            Speed:     55(+5)  Mysticism: 20(+0)
   Class:  Warrior                   Smarts:    30(+1)  Percept:   44(+2)
   Level:  12                    │ └─────────────────────────────────────────┘
   Exp:    4200/8000 (52%)        ┌─ .:Wealth ───────────┐ ┌─ .:Training ────┐
   Health: 142/200                │ Gold: 1,234          │ │ Train Pts: 5    │
 │ Mana:   80/100                 │ Bank: 10,000         │ │ Stat Pts:  2    │
 └───────────────────────────────┘└─────────────────────┘ └─────────────────┘
```

---

## API reference

### LoadPanelLayout

```go
func LoadPanelLayout(name string) (*PanelLayout, error)
```

Loads a layout from `<datafiles>/panel-layouts/<name>.yaml`. Returns an error
if the file does not exist or cannot be parsed. The returned layout has all
panels registered and ready to populate via `Panel(id)`.

```go
layout, err := templates.LoadPanelLayout("character/status")
if err != nil {
    tplTxt, _ := templates.Process("character/status", user, user.UserId)
    return tplTxt  // fall back to legacy template
}
```

---

### NewPanelLayout

```go
func NewPanelLayout(border, charset string, gap, margin int) *PanelLayout
```

Creates a blank layout entirely in Go. Use `AddSlot` and `AddPanelsToSlot` to
build the structure, then `Panel(id)` to configure and populate each panel.

| Parameter | Description |
| --- | --- |
| `border` | `"full"` or `"open"` |
| `charset` | `"single"`, `"double"`, or `"rounded"` |
| `gap` | Spaces between side-by-side panels and between slots |
| `margin` | Spaces prepended to every output line |

```go
layout := templates.NewPanelLayout("full", "rounded", 1, 1)
```

---

### PanelLayout.AddSlot

```go
func (l *PanelLayout) AddSlot() *LayoutSlot
```

Appends a new empty slot (vertical column) to the layout and returns it.
Slots are rendered side by side in the order they are added.

```go
leftSlot  := layout.AddSlot()
rightSlot := layout.AddSlot()
```

---

### PanelLayout.AddPanelsToSlot

```go
func (l *PanelLayout) AddPanelsToSlot(slot *LayoutSlot, ids ...string)
```

Appends a row of panels to `slot`. One panel is created per id, with default
settings. Use `Panel(id)` to configure each panel afterwards.

```go
// Left slot: one panel
layout.AddPanelsToSlot(leftSlot, "info")

// Right slot, first row: one panel
layout.AddPanelsToSlot(rightSlot, "attributes")

// Right slot, second row: two panels side by side
layout.AddPanelsToSlot(rightSlot, "wealth", "training")
```

---

### PanelLayout.Panel

```go
func (l *PanelLayout) Panel(id string) *Panel
```

Returns the panel with the given id. Panics with a descriptive message if the
id does not exist. This is intentional — a missing panel id is a programming
error that should surface immediately.

```go
layout.Panel("info").
    Add(`<ansi fg="yellow">Name:</ansi>`, `<ansi fg="yellow">N:</ansi>`, name)
```

---

### PanelLayout.Render

```go
func (l *PanelLayout) Render() string
```

Synthesizes the entire layout into a single terminal string. ANSI tags in
titles, labels, and values are preserved as-is. The string does not end with a
newline.

```go
user.SendText(layout.Render())
```

---

### Panel.Add

```go
func (p *Panel) Add(fullLabel, shortLabel, value string) *Panel
```

Appends a label+value row. The renderer uses `fullLabel` when it fits within
the panel's inner width; if not, it falls back to `shortLabel`. All three
strings may contain ANSI tags — visual width is always measured with tags
stripped. Returns the panel for chaining.

```go
layout.Panel("info").
    Add(`<ansi fg="yellow">Health:</ansi>`, `<ansi fg="yellow">HP:</ansi>`, hpString).
    Add(`<ansi fg="yellow">Mana:  </ansi>`, `<ansi fg="yellow">MP:</ansi>`, mpString)
```

---

### Panel.AddBlank

```go
func (p *Panel) AddBlank() *Panel
```

Appends an empty spacer row. Useful for visual grouping. Returns the panel for
chaining.

```go
layout.Panel("info").
    Add(`<ansi fg="yellow">Name:</ansi>`, `<ansi fg="yellow">N:</ansi>`, name).
    AddBlank().
    Add(`<ansi fg="yellow">Level:</ansi>`, `<ansi fg="yellow">Lvl:</ansi>`, level)
```

---

### Panel.SetCharset

```go
func (p *Panel) SetCharset(name string)
```

Overrides the border character set for this panel. When set, this panel uses
its own charset regardless of the layout-level setting. Recognised values:
`"single"`, `"double"`, `"rounded"`. An unrecognised value falls back to
`"single"`.

```go
layout.Panel("highlight").SetCharset("double")
```

---

### Panel.SetTitle

```go
func (p *Panel) SetTitle(title string)
```

Sets the title string rendered verbatim into the top border of the panel. May
contain ANSI tags. Visual width is measured with tags stripped.

The leading and trailing characters of the title string become the spacing
between the corner/horizontal characters and the title text. To match the
standard GoMud `.:` prefix:

```go
layout.Panel("info").SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Info</ansi> `)
```

To produce a plain title:

```go
layout.Panel("info").SetTitle(" Info ")
```

To produce a panel with no title:

```go
layout.Panel("info").SetTitle("")
```

---

### Panel.SetMinWidth

```go
func (p *Panel) SetMinWidth(w int)
```

Sets the minimum inner content width in terminal characters. The panel expands
automatically when any row's content is wider.

```go
layout.Panel("info").SetMinWidth(30)
```

---

### Panel.SetColumns

```go
func (p *Panel) SetColumns(n int)
```

Sets the number of label+value pairs per rendered line. Supported values are
`1` (default) and `2`. See [Multi-column panels](#multi-column-panels).

```go
layout.Panel("attributes").SetColumns(2)
```

---

### Panel.SetColumnGap

```go
func (p *Panel) SetColumnGap(n int)
```

Sets the number of spaces between the two columns when `columns` is `2`.
Defaults to `2`.

```go
layout.Panel("attributes").SetColumns(2).SetColumnGap(3)
```

---

## Width and ANSI tags

All width measurements strip ANSI tags before measuring with
`runewidth.StringWidth`. This means:

- ANSI colour tags in labels, values, and titles do not affect alignment.
- Multi-byte Unicode characters (e.g. CJK) count as their display width (2
  columns), not their byte length.

The renderer never truncates content. If a value is wider than the panel after
applying the short label, the panel expands to fit.

---

## Border styles

| Value | Behaviour |
| --- | --- |
| `"full"` | Every content row has side border characters: `│ content │` |
| `"open"` | Only the first and last content rows have side borders; interior rows are open: ` content ` |

Open borders give a lighter visual appearance and match the style of the
default GoMud status screen.

---

## Character sets

| Value | Top | Vertical | Bottom |
| --- | --- | --- | --- |
| `"single"` (default) | `┌─ title ───┐` | `│` | `└───────────┘` |
| `"double"` | `╔═ title ═══╗` | `║` | `╚═══════════╝` |
| `"rounded"` | `╭─ title ───╮` | `│` | `╰───────────╯` |

The charset applies to all panels in the layout. It is resolved by
`charsetForName(name string)` which is case-insensitive and defaults to
`"single"` for any unrecognised value.

---

## Multi-column panels

When `columns` is set to `2`, rows are paired during rendering:

- Rows 0 and 1 share a line (left and right columns).
- Rows 2 and 3 share the next line.
- An odd trailing row spans the full panel width.

The inner width is computed as `colWidth*2 + columnGap`, where `colWidth` is
the maximum of `(minWidth-columnGap)/2` and the widest single cell across all
rows.

```go
layout.Panel("attributes").
    SetColumns(2).
    SetColumnGap(2).
    Add(`Strength:`, `Str:`, "42").  // left column, line 1
    Add(`Vitality:`, `Vit:`, "38").  // right column, line 1
    Add(`Speed:`,    `Spd:`, "55").  // left column, line 2
    Add(`Mysticism:`,`Mys:`, "20")   // right column, line 2
```

Output:
```
│ Strength: 42  Vitality: 38 │
│ Speed:    55  Mysticism: 20 │
```

---

## Margin

`margin` prepends a fixed number of spaces to every output line. This is
equivalent to indenting the entire layout from the left edge of the terminal.
A value of `1` matches the single-space indent used by the standard GoMud
status screen.

Set via YAML:

```yaml
margin: 1
```

Or via `NewPanelLayout`:

```go
layout := templates.NewPanelLayout("full", "single", 1, 1) // last arg is margin
```
