package usercommands

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func buildJobsPanel(user *users.UserRecord) string {
	allRanks := user.Character.GetAllSkillRanks()

	type jobRow struct {
		name       string
		experience string
		barFull    string
		barEmpty   string
		completion string
	}

	rows := make([]jobRow, 0)
	for _, rank := range skills.GetProfessionRanks(allRanks) {
		barFull, barEmpty := util.ProgressBar(rank.Completion, 39)
		rows = append(rows, jobRow{
			name:       rank.Profession,
			experience: rank.ExperienceTitle,
			barFull:    barFull,
			barEmpty:   barEmpty,
			completion: fmt.Sprintf(`%d%%`, int(math.Floor(rank.Completion*100))),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.Compare(rows[i].name, rows[j].name) == -1
	})

	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "jobs")

	layout.Panel("jobs").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Jobs</ansi> `).
		SetMinWidth(74)

	if len(rows) == 0 {
		layout.Panel("jobs").Add(``, ``, `No jobs. Train skills to unlock professions.`)
		return layout.Render() + term.CRLFStr
	}

	maxNameWidth := 0
	for _, r := range rows {
		if w := len(r.name); w > maxNameWidth {
			maxNameWidth = w
		}
	}

	panel := layout.Panel("jobs").SetLabelWidth(maxNameWidth)
	for _, r := range rows {
		value := fmt.Sprintf(
			`<ansi fg="white-bold">%-11s</ansi> <ansi fg="green">%s</ansi><ansi fg="black-bold">%s</ansi> <ansi fg="cyan-bold">%s</ansi>`,
			r.experience, r.barFull, r.barEmpty, r.completion,
		)
		panel.Add(
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, r.name),
			fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, r.name),
			value,
		)
	}

	return layout.Render() + term.CRLFStr
}
