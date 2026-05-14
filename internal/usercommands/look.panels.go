package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// buildDescriptionPanel renders the character/description template as a panel.
// It accepts a *characters.Character and the viewing user's id.
func buildDescriptionPanel(c *characters.Character) string {
	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "desc")

	layout.Panel("desc").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Description</ansi> `).
		SetMinWidth(74)

	panel := layout.Panel("desc")

	for _, line := range util.SplitString(c.GetDescription(), 72) {
		panel.Add(``, ``, line)
	}
	panel.Add(``, ``, c.GetHealthAppearance())

	return layout.Render() + term.CRLFStr
}

// buildCorpseDescriptionPanel renders the character/description-corpse template as a panel.
func buildCorpseDescriptionPanel(c *characters.Character) string {
	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "desc")

	layout.Panel("desc").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Description</ansi> `).
		SetMinWidth(74)

	skulls := `<ansi fg="red-bold">☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠ ☠</ansi>`

	panel := layout.Panel("desc")
	panel.Add(``, ``, skulls)
	for _, line := range util.SplitString(c.GetDescription(), 72) {
		panel.Add(``, ``, fmt.Sprintf(`<ansi fg="8">%s</ansi>`, line))
	}
	panel.Add(``, ``, `<ansi fg="8">This is a corpse. They are dead.</ansi>`)
	panel.Add(``, ``, skulls)

	return layout.Render() + term.CRLFStr
}

// buildInventoryLookPanel renders the character/inventory-look template as a panel.
// equipment is *characters.Worn, itemNames is the list of carried item display names.
func buildInventoryLookPanel(equipment *characters.Worn, itemNames []string) string {
	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "equip")

	layout.Panel("equip").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Equipment</ansi> `).
		SetMinWidth(74).
		SetLabelWidth(9)

	panel := layout.Panel("equip")

	slots := []struct {
		label string
		item  func() *items.Item
	}{
		{`Weapon`, func() *items.Item { return &equipment.Weapon }},
		{`Offhand`, func() *items.Item { return &equipment.Offhand }},
		{`Head`, func() *items.Item { return &equipment.Head }},
		{`Neck`, func() *items.Item { return &equipment.Neck }},
		{`Body`, func() *items.Item { return &equipment.Body }},
		{`Belt`, func() *items.Item { return &equipment.Belt }},
		{`Gloves`, func() *items.Item { return &equipment.Gloves }},
		{`Ring`, func() *items.Item { return &equipment.Ring }},
		{`Legs`, func() *items.Item { return &equipment.Legs }},
		{`Feet`, func() *items.Item { return &equipment.Feet }},
	}

	for _, s := range slots {
		item := s.item()
		if item.IsDisabled() {
			continue
		}
		panel.Add(
			fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, s.label),
			fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, s.label),
			fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.NameSimple()),
		)
	}

	var sb strings.Builder
	itemCt := len(itemNames)
	switch {
	case itemCt == 0:
		sb.WriteString(`no`)
	case itemCt < 4:
		sb.WriteString(`a few`)
	case itemCt < 7:
		sb.WriteString(`several`)
	default:
		sb.WriteString(`lots of`)
	}
	sb.WriteString(` objects`)

	return layout.Render() + term.CRLFStr + ` Carrying: ` + sb.String() + term.CRLFStr
}

// buildPetPanel renders the character/pet template as a panel.
func buildPetPanel(c *characters.Character, isOwner bool) string {
	var out strings.Builder

	out.WriteString(fmt.Sprintf(`<ansi fg="black-bold">.:</ansi> %s`+term.CRLFStr, c.Pet.DisplayName()))

	// Description panel
	{
		layout := templates.NewPanelLayout("open", "single", 1, 1)
		slot := layout.AddSlot()
		layout.AddPanelsToSlot(slot, "desc")
		layout.Panel("desc").
			SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Description</ansi> `).
			SetMinWidth(74)

		panel := layout.Panel("desc")
		panel.Add(``, ``,
			fmt.Sprintf(`%s is a level <ansi fg="level">%d</ansi> pet <ansi fg="petname">%s</ansi> owned by <ansi fg="username">%s</ansi>.`,
				c.Pet.DisplayName(), c.Pet.Level, c.Pet.Type, c.Name),
		)
		panel.Add(``, ``,
			fmt.Sprintf(`%s is <ansi fg="hunger-%s">%s</ansi>`, c.Pet.DisplayName(), c.Pet.Food, c.Pet.Food),
		)
		out.WriteString(layout.Render() + term.CRLFStr)
	}

	// Abilities panel (owner only)
	if isOwner {
		if ab := c.Pet.GetCurrentAbilityDisplay(); ab != nil {
			layout := templates.NewPanelLayout("open", "single", 1, 1)
			slot := layout.AddSlot()
			layout.AddPanelsToSlot(slot, "abilities")
			layout.Panel("abilities").
				SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Abilities</ansi> `).
				SetMinWidth(74)

			panel := layout.Panel("abilities")

			if ab.CombatChance > 0 && ab.DiceRoll != `` {
				panel.Add(``, ``, `<ansi fg="yellow">Combat</ansi>`)
				panel.Add(``, ``,
					fmt.Sprintf(`%s will join you in combat <ansi fg="damage">%d%%</ansi> of the time for <ansi fg="damage">%dd%d</ansi>`,
						c.Pet.DisplayName(), ab.CombatChance, ab.DiceCount, ab.SideCount),
				)
			}

			if len(ab.StatMods) > 0 {
				panel.Add(``, ``, `<ansi fg="yellow">Stat Bonus</ansi>`)
				for stat, val := range ab.StatMods {
					panel.Add(``, ``, fmt.Sprintf(`<ansi fg="statmod">%s: %d</ansi>`, stat, val))
				}
			}

			if len(ab.BuffNames) > 0 {
				panel.Add(``, ``, `<ansi fg="yellow">Buffs</ansi>`)
				for _, name := range ab.BuffNames {
					panel.Add(``, ``, fmt.Sprintf(`<ansi fg="buff">%s</ansi>`, name))
				}
			}

			if ab.Capacity > 0 {
				panel.Add(``, ``, `<ansi fg="yellow">Carry</ansi>`)
				panel.Add(``, ``, fmt.Sprintf(`Capacity: <ansi fg="statmod">%d</ansi>`, ab.Capacity))
			}

			out.WriteString(layout.Render() + term.CRLFStr)
		}
	}

	// Carrying line
	out.WriteString(` Carrying: `)
	itemCt := len(c.Pet.Items)
	if itemCt == 0 {
		out.WriteString(`nothing`)
	} else {
		names := make([]string, 0, itemCt)
		for _, itm := range c.Pet.Items {
			names = append(names, itm.DisplayName())
		}
		out.WriteString(strings.Join(names, `, `))
	}
	out.WriteString(term.CRLFStr)

	return out.String()
}
