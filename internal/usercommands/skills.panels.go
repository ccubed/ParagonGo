package usercommands

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func buildSkillsPanel(user *users.UserRecord) string {
	allSkills := user.Character.GetSkills()

	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "skills")

	layout.Panel("skills").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Skills</ansi> `).
		SetMinWidth(74)

	if len(allSkills) == 0 {
		layout.Panel("skills").Add(``, ``, `No Skills! Visit a guild or training center to train.`)
		return layout.Render() + term.CRLFStr
	}

	names := make([]string, 0, len(allSkills))
	for name := range allSkills {
		names = append(names, name)
	}
	sort.Strings(names)

	maxNameWidth := 0
	for _, name := range names {
		if len(name) > maxNameWidth {
			maxNameWidth = len(name)
		}
	}

	panel := layout.Panel("skills").SetLabelWidth(maxNameWidth)
	for _, name := range names {
		level := allSkills[name]
		cooldown := user.Character.GetCooldown(name)

		var levelStr string
		switch level {
		case 0:
			levelStr = `[Unknown]`
		case 4:
			levelStr = `<ansi fg="white">[MAXIMUM]</ansi>`
		default:
			levelStr = fmt.Sprintf(`<ansi fg="white">[Level %d]</ansi>`, level)
		}

		var cdStr string
		if cooldown > 0 {
			cdStr = fmt.Sprintf(`  <ansi fg="red">cooling down for %d more round(s)</ansi>`, cooldown)
		}

		value := levelStr + cdStr

		panel.Add(
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, name),
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, name),
			value,
		)
	}

	return layout.Render() + term.CRLFStr
}
