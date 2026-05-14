package usercommands

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func buildCooldownsPanel(user *users.UserRecord) string {
	allCooldowns := user.Character.GetAllCooldowns()

	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "cooldowns")

	layout.Panel("cooldowns").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Cooldowns</ansi> `).
		SetMinWidth(74)

	if len(allCooldowns) == 0 {
		layout.Panel("cooldowns").Add(``, ``, `<ansi fg="black-bold">None</ansi>`)
		return layout.Render()
	}

	names := make([]string, 0, len(allCooldowns))
	for name := range allCooldowns {
		names = append(names, name)
	}
	sort.Strings(names)

	maxNameWidth := 0
	for _, name := range names {
		if len(name) > maxNameWidth {
			maxNameWidth = len(name)
		}
	}

	panel := layout.Panel("cooldowns").SetLabelWidth(maxNameWidth)
	roundSecs := int(configs.GetTimingConfig().RoundSeconds)
	for _, name := range names {
		rounds := allCooldowns[name]
		timeStr := formatDurationFromRounds(rounds, roundSecs)
		value := fmt.Sprintf(`<ansi fg="red">%d rounds</ansi> - %s`, rounds, timeStr)
		panel.Add(
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, name),
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, name),
			value,
		)
	}

	return layout.Render() + term.CRLFStr
}

// formatDurationFromRounds converts a round count to a human-readable duration
// string using the same logic as the roundstotime template function.
func formatDurationFromRounds(rounds, roundSeconds int) string {
	seconds := rounds * roundSeconds
	days := seconds / (24 * 3600)
	hours := (seconds % (24 * 3600)) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	result := ""
	if days == 1 {
		result += "1 day "
	} else if days > 1 {
		result += fmt.Sprintf("%d days ", days)
	}
	if hours > 0 || days > 0 {
		if hours == 1 {
			result += "1 hour "
		} else {
			result += fmt.Sprintf("%d hours ", hours)
		}
	}
	if minutes > 0 || hours > 0 || days > 0 {
		if minutes == 1 {
			result += "1 minute "
		} else {
			result += fmt.Sprintf("%d minutes ", minutes)
		}
	}
	if secs == 1 {
		result += "1 second"
	} else if secs > 0 {
		result += fmt.Sprintf("%d seconds", secs)
	}
	if result == "" {
		result = "0 seconds"
	}

	// trim trailing space
	for len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return result
}
