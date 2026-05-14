package usercommands

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type questRow struct {
	id          int
	name        string
	description string
	completion  float64
}

func buildQuestsPanel(rows []questRow, questsFound, questsTotal int) string {
	title := fmt.Sprintf(
		` <ansi fg="black-bold">.:</ansi><ansi fg="20">Quests ( %d out of %d shown )</ansi> `,
		questsFound, questsTotal,
	)

	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "quests")

	layout.Panel("quests").
		SetTitle(title).
		SetMinWidth(74)

	panel := layout.Panel("quests")

	if len(rows) == 0 {
		panel.Add(``, ``, `No quests in progress.`)
	} else {
		for i, r := range rows {
			barFull, barEmpty := util.ProgressBar(r.completion, 25)
			pct := fmt.Sprintf(`%d%%`, int(math.Floor(r.completion*100)))

			panel.Add(
				``, ``,
				fmt.Sprintf(
					`<ansi fg="questname">%-41s</ansi> <ansi fg="green">%s</ansi><ansi fg="black-bold">%s</ansi> <ansi fg="cyan-bold">%-4s</ansi>`,
					r.name, barFull, barEmpty, pct,
				),
			)
			panel.Add(``, ``, fmt.Sprintf(`<ansi fg="white-bold">%s</ansi>`, r.description))

			if i < len(rows)-1 {
				panel.AddBlank()
			}
		}
	}

	out := layout.Render() + term.CRLFStr

	if questsFound != questsTotal {
		out += ` <ansi fg="240">To see all quests (including completed), use <ansi fg="command">quests all</ansi></ansi>` + term.CRLFStr
	}

	return strings.TrimRight(out, " \t") + term.CRLFStr
}
