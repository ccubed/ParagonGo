package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func buildInspectPanel(inspectLevel int, itm *items.Item, iSpec *items.ItemSpec) string {
	var out strings.Builder

	// Basic Info panel
	{
		layout := templates.NewPanelLayout("open", "single", 1, 1)
		slot := layout.AddSlot()
		layout.AddPanelsToSlot(slot, "basic")
		layout.Panel("basic").
			SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Basic Info</ansi> `).
			SetMinWidth(74).SetLabelWidth(13)

		p := layout.Panel("basic")
		p.Add(`<ansi fg="yellow">Name:</ansi>`, `<ansi fg="yellow">Name:</ansi>`, strings.ToUpper(itm.Name()))
		descLines := util.SplitString(iSpec.Description, 60)
		for i, line := range descLines {
			label := ``
			if i == 0 {
				label = `<ansi fg="yellow">Description:</ansi>`
			}
			p.Add(label, label, line)
		}
		p.Add(`<ansi fg="yellow">Type:</ansi>`, `<ansi fg="yellow">Type:</ansi>`,
			fmt.Sprintf(`<ansi fg="white">%s</ansi> (<ansi fg="white">%s</ansi>)`, strings.ToUpper(iSpec.Type.String()), strings.ToUpper(iSpec.Subtype.String())))
		p.Add(`<ansi fg="yellow">Value:</ansi>`, `<ansi fg="yellow">Val:</ansi>`,
			fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, iSpec.Value))
		out.WriteString(layout.Render() + term.CRLFStr)
	}

	// Specific Stats panel
	{
		layout := templates.NewPanelLayout("open", "single", 1, 1)
		slot := layout.AddSlot()
		layout.AddPanelsToSlot(slot, "stats")
		layout.Panel("stats").
			SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Specific Stats</ansi> `).
			SetMinWidth(74).SetLabelWidth(13)

		p := layout.Panel("stats")
		if inspectLevel > 1 {
			if iSpec.Type == items.Weapon {
				damage := itm.GetDamage()
				p.Add(`<ansi fg="yellow">Damage:</ansi>`, `<ansi fg="yellow">Dmg:</ansi>`,
					fmt.Sprintf(`<ansi fg="damage">%s</ansi>`, util.FormatDiceRoll(damage.Attacks, damage.DiceCount, damage.SideCount, damage.BonusDamage, []int{})))
				if iSpec.Hands == items.TwoHanded {
					p.Add(`<ansi fg="yellow">Hands:</ansi>`, `<ansi fg="yellow">Hands:</ansi>`, `<ansi fg="white">Two-handed</ansi>`)
				} else {
					p.Add(`<ansi fg="yellow">Hands:</ansi>`, `<ansi fg="yellow">Hands:</ansi>`, `<ansi fg="white">One-handed</ansi>`)
				}
				if iSpec.WaitRounds > 0 {
					p.Add(`<ansi fg="yellow">Speed:</ansi>`, `<ansi fg="yellow">Speed:</ansi>`,
						fmt.Sprintf(`<ansi fg="red">Slow (+%d rounds per attack)</ansi>`, iSpec.WaitRounds))
				}
			} else {
				p.Add(`<ansi fg="yellow">Damage:</ansi>`, `<ansi fg="yellow">Dmg:</ansi>`, `<ansi fg="white">N/A</ansi>`)
			}
			if iSpec.DamageReduction > 0 {
				p.Add(`<ansi fg="yellow">Defense:</ansi>`, `<ansi fg="yellow">Def:</ansi>`,
					fmt.Sprintf(`<ansi fg="cyan">%d%% damage reduction</ansi>`, iSpec.DamageReduction))
			} else {
				p.Add(`<ansi fg="yellow">Defense:</ansi>`, `<ansi fg="yellow">Def:</ansi>`, `<ansi fg="white">N/A</ansi>`)
			}
			if iSpec.Uses > 0 {
				p.Add(`<ansi fg="yellow">Uses Left:</ansi>`, `<ansi fg="yellow">Uses:</ansi>`,
					fmt.Sprintf(`<ansi fg="white">%d</ansi>/<ansi fg="white">%d</ansi>`, itm.Uses, iSpec.Uses))
			}
			if iSpec.BreakChance > 0 {
				p.Add(`<ansi fg="yellow">Fragility:</ansi>`, `<ansi fg="yellow">Break:</ansi>`,
					fmt.Sprintf(`<ansi fg="red">%d%% chance to break on use</ansi>`, iSpec.BreakChance))
			}
		} else {
			p.Add(``, ``, `Unknown...`)
		}
		out.WriteString(layout.Render() + term.CRLFStr)
	}

	// Modifiers panel
	{
		layout := templates.NewPanelLayout("open", "single", 1, 1)
		slot := layout.AddSlot()
		layout.AddPanelsToSlot(slot, "mods")
		layout.Panel("mods").
			SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Modifiers</ansi> `).
			SetMinWidth(74).SetLabelWidth(13)

		p := layout.Panel("mods")
		if inspectLevel > 2 {
			added := false

			// Direct stat mods on the item spec (passive while equipped)
			for statName, qty := range iSpec.StatMods {
				var valStr string
				if qty >= 0 {
					valStr = fmt.Sprintf(`<ansi fg="green">+%d</ansi>`, qty)
				} else {
					valStr = fmt.Sprintf(`<ansi fg="red">%d</ansi>`, qty)
				}
				p.Add(
					fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, strings.ToUpper(statName+`:`)),
					fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, strings.ToUpper(statName+`:`)),
					valStr,
				)
				added = true
			}

			// Worn buffs: passive buffs active the entire time the item is equipped
			for _, buffId := range iSpec.WornBuffIds {
				spec := buffs.GetBuffSpec(buffId)
				if spec == nil {
					continue
				}
				name, _ := spec.VisibleNameDesc()
				p.Add(
					`<ansi fg="yellow">While Worn:</ansi>`,
					`<ansi fg="yellow">Worn:</ansi>`,
					fmt.Sprintf(`<ansi fg="spellname">%s</ansi>`, name),
				)
				for statName, qty := range spec.StatMods {
					var valStr string
					if qty >= 0 {
						valStr = fmt.Sprintf(`<ansi fg="green">+%d %s</ansi>`, qty, statName)
					} else {
						valStr = fmt.Sprintf(`<ansi fg="red">%d %s</ansi>`, qty, statName)
					}
					p.Add(``, ``, `  `+valStr)
				}
				for _, flag := range spec.Flags {
					allFlags := buffs.GetAllFlags()
					if desc, ok := allFlags[flag]; ok {
						p.Add(``, ``, fmt.Sprintf(`  <ansi fg="cyan">%s</ansi>`, desc))
					}
				}
				added = true
			}

			// On-use buffs: buffs applied when the item is used/consumed
			for _, buffId := range iSpec.BuffIds {
				spec := buffs.GetBuffSpec(buffId)
				if spec == nil {
					continue
				}
				duration := buffDurationString(spec)
				p.Add(
					`<ansi fg="yellow">On Use:</ansi>`,
					`<ansi fg="yellow">On Use:</ansi>`,
					fmt.Sprintf(`<ansi fg="spellname">%s</ansi> <ansi fg="white">- %s</ansi>`, spec.Name, duration),
				)
				for statName, qty := range spec.StatMods {
					var valStr string
					if qty >= 0 {
						valStr = fmt.Sprintf(`<ansi fg="green">+%d %s</ansi>`, qty, statName)
					} else {
						valStr = fmt.Sprintf(`<ansi fg="red">%d %s</ansi>`, qty, statName)
					}
					p.Add(``, ``, `  `+valStr)
				}
				for _, flag := range spec.Flags {
					allFlags := buffs.GetAllFlags()
					if desc, ok := allFlags[flag]; ok {
						p.Add(``, ``, fmt.Sprintf(`  <ansi fg="cyan">%s</ansi>`, desc))
					}
				}
				added = true
			}

			if !added {
				p.Add(``, ``, `None`)
			}
		} else {
			p.Add(``, ``, `Unknown...`)
		}
		out.WriteString(layout.Render() + term.CRLFStr)
	}

	// Magical Effects panel
	{
		layout := templates.NewPanelLayout("open", "single", 1, 1)
		slot := layout.AddSlot()
		layout.AddPanelsToSlot(slot, "magic")
		layout.Panel("magic").
			SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Magical Effects</ansi> `).
			SetMinWidth(74).SetLabelWidth(13)

		p := layout.Panel("magic")
		if inspectLevel > 3 {
			added := false
			if itm.IsCursed() {
				p.Add(``, ``, `It's <ansi fg="red-bold">CURSED!</ansi>`)
				added = true
			}
			if el := iSpec.Element.String(); len(el) > 0 {
				p.Add(`<ansi fg="yellow">Element:</ansi>`, `<ansi fg="yellow">Elem:</ansi>`,
					fmt.Sprintf(`<ansi fg="cyan">%s</ansi>`, strings.ToUpper(el)))
				added = true
			}
			for _, buffId := range iSpec.Damage.CritBuffIds {
				spec := buffs.GetBuffSpec(buffId)
				if spec == nil {
					continue
				}
				duration := buffDurationString(spec)
				p.Add(
					`<ansi fg="yellow">Crits Apply:</ansi>`,
					`<ansi fg="yellow">Crit:</ansi>`,
					fmt.Sprintf(`<ansi fg="spellname">%s</ansi> - %s`, spec.Name, duration),
				)
				added = true
			}
			if !added {
				p.Add(``, ``, `None`)
			}
		} else {
			p.Add(``, ``, `Unknown...`)
		}
		out.WriteString(layout.Render() + term.CRLFStr)
	}

	return out.String()
}

func buffDurationString(spec *buffs.BuffSpec) string {
	if spec.RoundInterval == 1 && spec.TriggerCount == 1 {
		return `Activates once`
	}
	roundCt := `round`
	if spec.RoundInterval > 1 {
		roundCt = fmt.Sprintf(`%d rounds`, spec.RoundInterval)
	}
	return fmt.Sprintf(`Activates every %s (%dx total)`, roundCt, spec.TriggerCount)
}

func buildTrackPanel(visitors []trackingInfo) string {
	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "track")
	layout.Panel("track").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Recent Visitors</ansi> `).
		SetMinWidth(74)

	p := layout.Panel("track")
	if len(visitors) == 0 {
		p.Add(``, ``, `None`)
	} else {
		for _, v := range visitors {
			name := v.Name
			if name == `` {
				name = `None`
			}
			strength := strings.ToLower(v.Strength)
			label := fmt.Sprintf(`[<ansi fg="trail-%s">%s</ansi>]`, strength, v.Strength)
			value := fmt.Sprintf(`<ansi fg="username">%s</ansi>`, name)
			if v.ExitName != `` {
				value += fmt.Sprintf(` - It seems like they went <ansi fg="exit">%s</ansi>`, v.ExitName)
			}
			p.Add(label, label, value)
		}
	}

	return layout.Render() + term.CRLFStr
}

func buildRoomDescPanel(details rooms.RoomTemplateDetails) string {
	var out strings.Builder

	descColor := `room-description`
	if details.IsNight || details.IsDark {
		descColor = `room-description-dark`
	}
	out.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, descColor, details.Description))

	for _, alert := range details.RoomAlerts {
		out.WriteString(term.CRLFStr)
		out.WriteString(term.CRLFStr)
		out.WriteString(`    <ansi fg="red">┌───────────────────────────────────────────────────────────────────┐</ansi>`)
		out.WriteString(term.CRLFStr)
		out.WriteString(`      ` + alert)
		out.WriteString(term.CRLFStr)
		out.WriteString(`    <ansi fg="red">└───────────────────────────────────────────────────────────────────┘</ansi>`)
	}

	if details.TrackingString != `` {
		out.WriteString(term.CRLFStr)
		out.WriteString(fmt.Sprintf(`<ansi fg="182">%s</ansi>`, details.TrackingString))
		out.WriteString(term.CRLFStr)
	}

	return out.String()
}

func buildInsideContainerPanel(itemNames []string, itemNamesFormatted []string) string {
	layout := templates.NewPanelLayout("open", "single", 1, 0)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "inside")
	layout.Panel("inside").SetMinWidth(74)

	p := layout.Panel("inside")
	if len(itemNames) == 0 {
		p.Add(`<ansi fg="white">Inside:</ansi>`, `<ansi fg="white">Inside:</ansi>`, `Nothing`)
	} else {
		// Wrap item names into lines of ~66 chars
		var line strings.Builder
		lineLen := 0
		first := true
		label := `<ansi fg="white">Inside:</ansi>`
		for i, name := range itemNames {
			proposed := lineLen + len(name) + 2
			if !first && proposed > 66 {
				p.Add(label, label, line.String())
				label = ``
				line.Reset()
				lineLen = 0
			}
			if line.Len() > 0 {
				line.WriteString(`, `)
				lineLen += 2
			}
			line.WriteString(itemNamesFormatted[i])
			lineLen += len(name)
			first = false
		}
		if line.Len() > 0 {
			p.Add(label, label, line.String())
		}
	}

	return layout.Render() + term.CRLFStr
}

func buildTrainPanel(data TrainingOptions) string {
	var out strings.Builder

	out.WriteString(`Train here to pick up new and interesting skills. You can train skills more than once to increase their effectiveness.` + term.CRLFStr)
	out.WriteString(term.CRLFStr)
	out.WriteString(`Type "<ansi fg="command">help [skill_name]</ansi>" to find out more` + term.CRLFStr)
	out.WriteString(term.CRLFStr)

	layout := templates.NewPanelLayout("open", "single", 1, 1)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "train")
	layout.Panel("train").
		SetTitle(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Skills Taught Here</ansi> `).
		SetMinWidth(74).SetLabelWidth(12)

	p := layout.Panel("train")
	for _, opt := range data.Options {
		var nameStr string
		if opt.Cost == 0 {
			nameStr = fmt.Sprintf(`<ansi fg="white">%s</ansi>`, opt.Name)
		} else {
			nameStr = fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi>`, opt.Name)
		}
		value := fmt.Sprintf(`<ansi fg="white">[%-7s]</ansi> <ansi fg="white">%s</ansi>`, opt.CurrentStatus, opt.Message)
		p.Add(nameStr, nameStr, value)
	}

	out.WriteString(layout.Render() + term.CRLFStr)

	ptColor := `yellow`
	if data.TrainingPoints == 0 {
		ptColor = `red`
	}
	out.WriteString(fmt.Sprintf(`  You have <ansi fg="%s-bold">%d Training Points</ansi> to spend. Level up to earn more.`, ptColor, data.TrainingPoints) + term.CRLFStr)
	out.WriteString(term.CRLFStr)
	out.WriteString(`To train a skill, type "<ansi fg="command">train [skill_name]</ansi>"` + term.CRLFStr)

	return out.String()
}

func buildBiomePanel(biome *rooms.BiomeInfo) string {
	layout := templates.NewPanelLayout("open", "single", 1, 0)
	slot := layout.AddSlot()
	layout.AddPanelsToSlot(slot, "biome")
	layout.Panel("biome").
		SetTitle(` <ansi fg="black-bold">.:</ansi> Biome Info `).
		SetMinWidth(74).SetLabelWidth(13)

	var lighting string
	if biome.IsDark() {
		lighting = `It's always dark.`
	} else if biome.IsLit() {
		lighting = `It is kept well lit at night.`
	} else {
		lighting = `Visibility is affected by the day/night cycle.`
	}

	p := layout.Panel("biome")
	p.Add(`<ansi fg="yellow">Name:</ansi>`, `<ansi fg="yellow">Name:</ansi>`, biome.Name)
	p.Add(`<ansi fg="yellow">Symbol:</ansi>`, `<ansi fg="yellow">Sym:</ansi>`, biome.SymbolString())
	p.Add(`<ansi fg="yellow">Lighting:</ansi>`, `<ansi fg="yellow">Light:</ansi>`, lighting)
	biomeDescLines := util.SplitString(biome.Description, 60)
	for i, line := range biomeDescLines {
		label := ``
		if i == 0 {
			label = `<ansi fg="yellow">Description:</ansi>`
		}
		p.Add(label, label, line)
	}

	return layout.Render() + term.CRLFStr
}
