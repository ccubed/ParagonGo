package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func buildConditionsPanel(user *users.UserRecord) string {
	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "conditions")

	layout.Panel("conditions").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Conditions</ansi> `).
		SetMinWidth(74)

	charBuffs := user.Character.GetBuffs()
	if len(charBuffs) == 0 {
		layout.Panel("conditions").Add(``, ``, `<ansi fg="black-bold">None</ansi>`)
		return layout.Render() + term.CRLFStr
	}

	// Collect rows and measure the longest name for label alignment.
	type condRow struct {
		name        string
		description string
		permaBuff   bool
		roundsLeft  int
	}
	rows := make([]condRow, 0, len(charBuffs))
	maxNameWidth := 0
	roundSecs := int(configs.GetTimingConfig().RoundSeconds)

	for _, buff := range charBuffs {
		spec := buffs.GetBuffSpec(buff.BuffId)
		_, roundsLeft := buffs.GetDurations(buff, spec)
		name, description := spec.VisibleNameDesc()
		rows = append(rows, condRow{
			name:        name,
			description: description,
			permaBuff:   buff.PermaBuff,
			roundsLeft:  roundsLeft,
		})
		if w := len(name); w > maxNameWidth {
			maxNameWidth = w
		}
	}

	panel := layout.Panel("conditions").SetLabelWidth(maxNameWidth)
	for _, row := range rows {
		var value string
		if row.permaBuff || row.roundsLeft >= buffs.TriggersLeftUnlimited {
			value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, row.description)
		} else {
			timeStr := formatDurationFromRounds(row.roundsLeft, roundSecs)
			value = fmt.Sprintf(`<ansi fg="yellow">%s</ansi>  <ansi fg="red">(%s left)</ansi>`, row.description, timeStr)
		}
		panel.Add(
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, row.name),
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, row.name),
			value,
		)
	}

	return layout.Render() + term.CRLFStr
}
