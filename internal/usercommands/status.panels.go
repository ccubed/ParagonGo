package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Used in `status`
func buildStatusPanel(user *users.UserRecord) string {
	c := user.Character

	allRanks := c.GetAllSkillRanks()
	profession := skills.GetProfession(allRanks)
	realXPNow, realXPTNL := c.XPTNLActual()
	xpPct := 0
	if realXPTNL > 0 {
		xpPct = (realXPNow * 100) / realXPTNL
	}
	xpValue := fmt.Sprintf(`%d/%d (%d%%)`, realXPNow, realXPTNL, xpPct)

	hLevel := util.QuantizeTens(c.Health, c.HealthMax.Value)
	hpValue := fmt.Sprintf(`<ansi fg="health-%d">%d</ansi>/<ansi fg="health-%d">%d</ansi>`,
		hLevel, c.Health, hLevel, c.HealthMax.Value)

	mLevel := util.QuantizeTens(c.Mana, c.ManaMax.Value)
	mpValue := fmt.Sprintf(`<ansi fg="mana-%d">%d</ansi>/<ansi fg="mana-%d">%d</ansi>`,
		mLevel, c.Mana, mLevel, c.ManaMax.Value)

	armorValue := fmt.Sprintf(`%d`, c.GetDefense())
	if bool(configs.GetGamePlayConfig().Death.PermaDeath) {
		armorValue += fmt.Sprintf(`  <ansi fg="yellow">Lives:</ansi> %d`, c.ExtraLives)
	}

	layout, err := templates.LoadPanelLayout("character/status")
	if err != nil {
		tplTxt, _ := templates.Process("character/status", user, user.UserId)
		return tplTxt
	}

	layout.Panel("info").
		Add(`<ansi fg="yellow">Area:   </ansi>`, `<ansi fg="yellow">Loc:</ansi>`, c.Zone).
		Add(`<ansi fg="yellow">Race:   </ansi>`, `<ansi fg="yellow">Rce:</ansi>`, fmt.Sprintf(`%s (%s)`, c.Race(), c.RaceSize())).
		Add(`<ansi fg="yellow">Level:  </ansi>`, `<ansi fg="yellow">Lvl:</ansi>`, fmt.Sprintf(`%d`, c.Level)).
		Add(`<ansi fg="yellow">Exp:    </ansi>`, `<ansi fg="yellow">XP: </ansi>`, xpValue).
		Add(`<ansi fg="yellow">Health: </ansi>`, `<ansi fg="yellow">HP: </ansi>`, hpValue).
		Add(`<ansi fg="yellow">Mana:   </ansi>`, `<ansi fg="yellow">MP: </ansi>`, mpValue).
		Add(`<ansi fg="yellow">Armor:  </ansi>`, `<ansi fg="yellow">Arm:</ansi>`, armorValue)

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

	layout.Panel("wealth").
		Add(`<ansi fg="yellow">Gold:</ansi>`, `<ansi fg="yellow">G:</ansi>`, util.FormatNumber(c.Gold)).
		Add(`<ansi fg="yellow">Bank:</ansi>`, `<ansi fg="yellow">B:</ansi>`, util.FormatNumber(c.Bank))

	layout.Panel("training").
		Add(`<ansi fg="yellow">Train Pts:</ansi>`, `<ansi fg="yellow">Trn:</ansi>`, fmt.Sprintf(`%d`, c.TrainingPoints)).
		Add(`<ansi fg="yellow">Stat Pts: </ansi>`, `<ansi fg="yellow">Sta:</ansi>`, fmt.Sprintf(`%d`, c.StatPoints))

	header := fmt.Sprintf(` <ansi fg="black-bold">.:</ansi> <ansi fg="username">%s</ansi> the <ansi fg="%s">%s</ansi> %s`,
		c.Name, c.AlignmentName(), c.AlignmentName(), profession)

	return header + term.CRLFStr + layout.Render()
}

// Used in `status bonuses`
func statusBonuses(user *users.UserRecord) (bool, error) {

	var sb strings.Builder

	sb.WriteString(term.CRLFStr)
	sb.WriteString(` <ansi fg="black-bold">.:</ansi><ansi fg="20">Stat Bonuses for </ansi><ansi fg="username">`)
	sb.WriteString(user.Character.Name)
	sb.WriteString(`</ansi>`)
	sb.WriteString(term.CRLFStr)

	type slotItem struct {
		SlotName string
		Item     items.Item
	}

	slots := []slotItem{
		{`Weapon`, user.Character.Equipment.Weapon},
		{`Offhand`, user.Character.Equipment.Offhand},
		{`Head`, user.Character.Equipment.Head},
		{`Neck`, user.Character.Equipment.Neck},
		{`Body`, user.Character.Equipment.Body},
		{`Belt`, user.Character.Equipment.Belt},
		{`Gloves`, user.Character.Equipment.Gloves},
		{`Ring`, user.Character.Equipment.Ring},
		{`Legs`, user.Character.Equipment.Legs},
		{`Feet`, user.Character.Equipment.Feet},
	}

	sb.WriteString(term.CRLFStr)
	sb.WriteString(` ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Equipment Bonuses</ansi> ──────────────────────────────────────────────────────┐`)

	equipFound := false
	for _, slot := range slots {
		if slot.Item.ItemId < 1 {
			continue
		}
		spec := slot.Item.GetSpec()
		if len(spec.StatMods) == 0 {
			continue
		}
		equipFound = true
		sb.WriteString(term.CRLFStr)
		sb.WriteString(fmt.Sprintf(`   <ansi fg="yellow">%-8s</ansi> %s`, slot.SlotName+`:`, slot.Item.DisplayName()))
		sb.WriteString(term.CRLFStr)
		sb.WriteString(`            `)
		sb.WriteString(formatStatMods(spec.StatMods))
	}

	if !equipFound {
		sb.WriteString(term.CRLFStr)
		sb.WriteString(`   <ansi fg="black-bold">None</ansi>`)
	}

	sb.WriteString(term.CRLFStr)
	sb.WriteString(` └────────────────────────────────────────────────────────────────────────────┘`)

	activeBuffs := user.Character.Buffs.GetBuffs()
	sb.WriteString(term.CRLFStr)
	sb.WriteString(` ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Buff Bonuses</ansi> ───────────────────────────────────────────────────────────┐`)

	buffFound := false
	for _, b := range activeBuffs {
		spec := buffs.GetBuffSpec(b.BuffId)
		if spec == nil || spec.Secret || len(spec.StatMods) == 0 {
			continue
		}
		buffFound = true
		sb.WriteString(term.CRLFStr)
		sb.WriteString(fmt.Sprintf(`   <ansi fg="yellow-bold">%s</ansi>`, spec.Name))
		sb.WriteString(term.CRLFStr)
		sb.WriteString(`            `)
		sb.WriteString(formatStatMods(spec.StatMods))
	}

	if !buffFound {
		sb.WriteString(term.CRLFStr)
		sb.WriteString(`   <ansi fg="black-bold">None</ansi>`)
	}

	sb.WriteString(term.CRLFStr)
	sb.WriteString(` └────────────────────────────────────────────────────────────────────────────┘`)

	sb.WriteString(term.CRLFStr)
	sb.WriteString(` ┌─ <ansi fg="black-bold">.:</ansi><ansi fg="20">Pet Bonuses</ansi> ────────────────────────────────────────────────────────────┐`)

	if user.Character.Pet.Exists() {
		mods := user.Character.Pet.GetEffectiveStatMods()
		if len(mods) > 0 {
			sb.WriteString(term.CRLFStr)
			sb.WriteString(fmt.Sprintf(`   %s`, user.Character.Pet.DisplayName()))
			sb.WriteString(term.CRLFStr)
			sb.WriteString(`            `)
			sb.WriteString(formatStatMods(mods))
		} else {
			sb.WriteString(term.CRLFStr)
			sb.WriteString(fmt.Sprintf(`   %s <ansi fg="black-bold">(no stat bonuses)</ansi>`, user.Character.Pet.DisplayName()))
		}
	} else {
		sb.WriteString(term.CRLFStr)
		sb.WriteString(`   <ansi fg="black-bold">No pet</ansi>`)
	}

	sb.WriteString(term.CRLFStr)
	sb.WriteString(` └────────────────────────────────────────────────────────────────────────────┘`)

	sb.WriteString(term.CRLFStr)
	sb.WriteString(term.CRLFStr)
	sb.WriteString(` <ansi fg="yellow">Total Stat Modifiers:</ansi>`)

	allStatNames := []string{`strength`, `speed`, `smarts`, `vitality`, `mysticism`, `perception`, `healthmax`, `manamax`, `healthrecovery`, `manarecovery`, `attacks`, `damage`}
	displayNames := map[string]string{
		`strength`: `Strength`, `speed`: `Speed`, `smarts`: `Smarts`,
		`vitality`: `Vitality`, `mysticism`: `Mysticism`, `perception`: `Perception`,
		`healthmax`: `Health Max`, `manamax`: `Mana Max`,
		`healthrecovery`: `HP Recovery`, `manarecovery`: `MP Recovery`,
		`attacks`: `Attacks`, `damage`: `Damage`,
	}

	totalFound := false
	for _, stat := range allStatNames {
		total := user.Character.StatMod(stat)
		if total == 0 {
			continue
		}
		totalFound = true
		color := `green`
		sign := `+`
		if total < 0 {
			color = `red`
			sign = ``
		}
		sb.WriteString(term.CRLFStr)
		sb.WriteString(fmt.Sprintf(`   <ansi fg="yellow">%-12s</ansi> <ansi fg="%s">%s%d</ansi>`, displayNames[stat]+`:`, color, sign, total))
	}

	if !totalFound {
		sb.WriteString(term.CRLFStr)
		sb.WriteString(`   <ansi fg="black-bold">None</ansi>`)
	}

	sb.WriteString(term.CRLFStr)

	user.SendText(sb.String())

	return true, nil
}

func formatStatMods(mods map[string]int) string {

	displayNames := map[string]string{
		`strength`: `Str`, `speed`: `Spd`, `smarts`: `Smt`,
		`vitality`: `Vit`, `mysticism`: `Mys`, `perception`: `Per`,
		`healthmax`: `HP`, `manamax`: `MP`,
		`healthrecovery`: `HP Rec`, `manarecovery`: `MP Rec`,
		`attacks`: `Atk`, `damage`: `Dmg`,
		`casting`: `Cast`, `xpscale`: `XP%`,
		`picklock`: `Pick`, `tame`: `Tame`,
	}

	keys := make([]string, 0, len(mods))
	for k := range mods {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := []string{}
	for _, k := range keys {
		v := mods[k]
		if v == 0 {
			continue
		}
		name := displayNames[k]
		if name == `` {
			if strings.HasPrefix(k, `casting-`) {
				name = `Cast:` + strings.TrimPrefix(k, `casting-`)
			} else if strings.HasPrefix(k, `racial-bonus-`) {
				name = `vs ` + strings.TrimPrefix(k, `racial-bonus-`)
			} else {
				name = k
			}
		}
		color := `green`
		sign := `+`
		if v < 0 {
			color = `red`
			sign = ``
		}
		parts = append(parts, fmt.Sprintf(`<ansi fg="%s">%s%d %s</ansi>`, color, sign, v, name))
	}

	return strings.Join(parts, `  `)
}
