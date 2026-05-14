package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// buildPeepPanel renders the status-lite view for a peep target.
// It accepts a *characters.Character so it works for both players and mobs.
func buildPeepPanel(c *characters.Character) string {
	layout, err := templates.LoadPanelLayout("character/status-lite")
	if err != nil {
		layout = buildPeepLayoutInline()
	}

	hLevel := util.QuantizeTens(c.Health, c.HealthMax.Value)
	hpValue := fmt.Sprintf(`<ansi fg="health-%d">%d</ansi>/<ansi fg="health-%d">%d</ansi>`,
		hLevel, c.Health, hLevel, c.HealthMax.Value)

	mLevel := util.QuantizeTens(c.Mana, c.ManaMax.Value)
	mpValue := fmt.Sprintf(`<ansi fg="mana-%d">%d</ansi>/<ansi fg="mana-%d">%d</ansi>`,
		mLevel, c.Mana, mLevel, c.ManaMax.Value)

	layout.Panel("info").
		Add(`<ansi fg="yellow">Race:   </ansi>`, `<ansi fg="yellow">Rce:</ansi>`, fmt.Sprintf(`%s (%s)`, c.Race(), c.RaceSize())).
		Add(`<ansi fg="yellow">Health: </ansi>`, `<ansi fg="yellow">HP: </ansi>`, hpValue).
		Add(`<ansi fg="yellow">Mana:   </ansi>`, `<ansi fg="yellow">MP: </ansi>`, mpValue).
		Add(`<ansi fg="yellow">Armor:  </ansi>`, `<ansi fg="yellow">Arm:</ansi>`, fmt.Sprintf(`%d`, c.GetDefense())).
		Add(`<ansi fg="yellow">Level:  </ansi>`, `<ansi fg="yellow">Lvl:</ansi>`, fmt.Sprintf(`%d`, c.Level)).
		Add(`<ansi fg="yellow">Gold:   </ansi>`, `<ansi fg="yellow">G:  </ansi>`, util.FormatNumber(c.Gold))

	addStat := func(fullLabel, shortLabel string, base, mod int) {
		value := fmt.Sprintf(`<ansi fg="stat">%-4d</ansi><ansi fg="statmod">(%-3d)</ansi>`, base, mod)
		layout.Panel("attributes").Add(fullLabel, shortLabel, value)
	}
	addStat(`<ansi fg="yellow">Strength:  </ansi>`, `<ansi fg="yellow">Str:</ansi>`, c.Stats.Strength.Value, c.StatMod("strength"))
	addStat(`<ansi fg="yellow">Speed:     </ansi>`, `<ansi fg="yellow">Spd:</ansi>`, c.Stats.Speed.Value, c.StatMod("speed"))
	addStat(`<ansi fg="yellow">Smarts:    </ansi>`, `<ansi fg="yellow">Smt:</ansi>`, c.Stats.Smarts.Value, c.StatMod("smarts"))
	addStat(`<ansi fg="yellow">Vitality:  </ansi>`, `<ansi fg="yellow">Vit:</ansi>`, c.Stats.Vitality.Value, c.StatMod("vitality"))
	addStat(`<ansi fg="yellow">Mysticism: </ansi>`, `<ansi fg="yellow">Mys:</ansi>`, c.Stats.Mysticism.Value, c.StatMod("mysticism"))
	addStat(`<ansi fg="yellow">Percept:   </ansi>`, `<ansi fg="yellow">Per:</ansi>`, c.Stats.Perception.Value, c.StatMod("perception"))

	allRanks := c.GetAllSkillRanks()
	profession := skills.GetProfession(allRanks)
	header := fmt.Sprintf(` <ansi fg="black-bold">.:</ansi> <ansi fg="username">%s</ansi> the <ansi fg="%s">%s</ansi> %s`,
		c.Name, c.AlignmentName(), c.AlignmentName(), profession)

	return header + term.CRLFStr + layout.Render()
}

// buildPeepLayoutInline constructs the status-lite layout in code as a fallback
// when the YAML file is not present.
func buildPeepLayoutInline() *templates.PanelLayout {
	layout := templates.NewPanelLayout("open", "single", 1, 1)
	left := layout.AddSlot()
	layout.AddPanelsToSlot(left, "info")
	right := layout.AddSlot()
	layout.AddPanelsToSlot(right, "attributes")

	layout.Panel("info").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Info</ansi> `).
		SetMinWidth(30)

	layout.Panel("attributes").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Attributes</ansi> `).
		SetMinWidth(42).
		SetColumns(2).
		SetColumnGap(2)

	return layout
}
