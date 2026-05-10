package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// colorNum wraps a numeric string in green (positive), red (negative), or
// yellow (zero) ANSI tags.
func colorNum(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] == '-' {
		return `<ansi fg="red">` + s + `</ansi>`
	}
	if s == "0" || s == "0.0" || s == "0.00" || s == "0.000" {
		return `<ansi fg="yellow">` + s + `</ansi>`
	}
	return `<ansi fg="green">` + s + `</ansi>`
}

/*
* Role Permissions:
* rankings 				(All)
 */
func Rankings(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := strings.Fields(rest)

	if len(args) == 0 {
		infoOutput, _ := templates.Process("admincommands/help/command.rankings", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	subCmd := strings.ToLower(args[0])

	switch subCmd {
	case "pets":
		return rankings_Pets(args[1:], user)
	case "weapons":
		return rankings_Weapons(args[1:], user)
	case "armor":
		return rankings_Armor(args[1:], user)
	default:
		infoOutput, _ := templates.Process("admincommands/help/command.rankings", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}
}

func rankings_Pets(args []string, user *users.UserRecord) (bool, error) {

	mode := "overall"
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "combat", "utility", "overall":
			mode = strings.ToLower(args[0])
		default:
			user.SendText(fmt.Sprintf(`Unknown sort mode "<ansi fg="red">%s</ansi>". Valid options: <ansi fg="command">combat</ansi>, <ansi fg="command">utility</ansi>, <ansi fg="command">overall</ansi>`, args[0]))
			return true, nil
		}
	}

	byCombat, byUtility, byOverall := combat.RankPets()

	var ranked []combat.PetRank
	var scoreLabel string
	var titleSuffix string

	switch mode {
	case "combat":
		ranked = byCombat
		scoreLabel = "Combat"
		titleSuffix = "Combat Score"
	case "utility":
		ranked = byUtility
		scoreLabel = "Utility"
		titleSuffix = "Utility Score"
	default:
		ranked = byOverall
		scoreLabel = "Overall"
		titleSuffix = "Overall Score"
	}

	headers := []string{"Rank", "Type", "PeakDPS", "PeakStats", "PeakCap", "Buffs", "BuffVal", scoreLabel}
	rows := make([][]string, 0, len(ranked))
	formatting := make([][]string, 0, len(ranked))

	for i, r := range ranked {
		var score string
		switch mode {
		case "combat":
			score = colorNum(fmt.Sprintf("%.3f", r.CombatScore))
		case "utility":
			score = colorNum(fmt.Sprintf("%.3f", r.UtilityScore))
		default:
			score = colorNum(fmt.Sprintf("%.3f", r.OverallScore))
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Type,
			colorNum(fmt.Sprintf("%.2f", r.PeakDPS)),
			colorNum(fmt.Sprintf("%d", r.PeakStatTotal)),
			colorNum(fmt.Sprintf("%d", r.PeakCapacity)),
			colorNum(fmt.Sprintf("%d", r.PeakBuffCount)),
			colorNum(fmt.Sprintf("%d", r.PeakBuffValue)),
			score,
		})
		formatting = append(formatting, []string{
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="petname">%s</ansi>`,
			`%s`, `%s`, `%s`, `%s`, `%s`, `%s`,
		})
	}

	tblData := templates.GetTable(fmt.Sprintf("Pet Rankings by %s", titleSuffix), headers, rows, formatting...)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}

func rankings_Weapons(args []string, user *users.UserRecord) (bool, error) {

	mode := "dps"
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "dps", "dps-adj", "max":
			mode = strings.ToLower(args[0])
		default:
			user.SendText(fmt.Sprintf(`Unknown sort mode "<ansi fg="red">%s</ansi>". Valid options: <ansi fg="command">dps</ansi>, <ansi fg="command">dps-adj</ansi>, <ansi fg="command">max</ansi>`, args[0]))
			return true, nil
		}
	}

	byDPS, byAdjDPS, byMaxDmg := combat.RankWeapons()

	var ranked []combat.WeaponRank
	var scoreLabel string
	var titleSuffix string

	switch mode {
	case "dps-adj":
		ranked = byAdjDPS
		scoreLabel = "AdjDPS"
		titleSuffix = "Adjusted DPS"
	case "max":
		ranked = byMaxDmg
		scoreLabel = "MaxDmg"
		titleSuffix = "Max Damage"
	default:
		ranked = byDPS
		scoreLabel = "DPS"
		titleSuffix = "DPS"
	}

	headers := []string{"Rank", "Name", "Dice", "Subtype", "Hands", "Wait", scoreLabel}
	rows := make([][]string, 0, len(ranked))
	formatting := make([][]string, 0, len(ranked))

	for i, r := range ranked {
		var score string
		switch mode {
		case "dps-adj":
			score = colorNum(fmt.Sprintf("%.3f", r.AdjDPS))
		case "max":
			score = fmt.Sprintf(`<ansi fg="green">%d</ansi> <ansi fg="yellow">(avg %.2f)</ansi>`, r.MaxDmg, r.AvgDmg)
		default:
			score = colorNum(fmt.Sprintf("%.3f", r.DPS))
		}

		waitStr := fmt.Sprintf("%d", r.WaitRounds)
		if r.WaitRounds > 0 {
			waitStr = `<ansi fg="red">` + waitStr + `</ansi>`
		} else {
			waitStr = `<ansi fg="yellow">` + waitStr + `</ansi>`
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Name,
			r.DiceRoll,
			string(r.Subtype),
			fmt.Sprintf("%d", r.Hands),
			waitStr,
			score,
		})
		formatting = append(formatting, []string{
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="itemname">%s</ansi>`,
			`<ansi fg="cyan">%s</ansi>`,
			`<ansi fg="white">%s</ansi>`,
			`<ansi fg="white">%s</ansi>`,
			`%s`,
			`%s`,
		})
	}

	tblData := templates.GetTable(fmt.Sprintf("Weapon Rankings by %s", titleSuffix), headers, rows, formatting...)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}

func rankings_Armor(args []string, user *users.UserRecord) (bool, error) {

	mode := "def"
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "def", "def-adj", "score":
			mode = strings.ToLower(args[0])
		default:
			user.SendText(fmt.Sprintf(`Unknown sort mode "<ansi fg="red">%s</ansi>". Valid options: <ansi fg="command">def</ansi>, <ansi fg="command">def-adj</ansi>, <ansi fg="command">score</ansi>`, args[0]))
			return true, nil
		}
	}

	byDefense, byAdjDefense, byScore := combat.RankArmor()

	var ranked []combat.ArmorRank
	var scoreLabel string
	var titleSuffix string

	switch mode {
	case "def-adj":
		ranked = byAdjDefense
		scoreLabel = "AdjDef"
		titleSuffix = "Adjusted Defense"
	case "score":
		ranked = byScore
		scoreLabel = "Score"
		titleSuffix = "Score"
	default:
		ranked = byDefense
		scoreLabel = "Defense"
		titleSuffix = "Defense"
	}

	headers := []string{"Rank", "Name", "Slot", "Cursed", "Buffs", "Stats", scoreLabel}
	rows := make([][]string, 0, len(ranked))
	formatting := make([][]string, 0, len(ranked))

	for i, r := range ranked {
		var cursed string
		if r.Cursed {
			cursed = `<ansi fg="red">yes</ansi>`
		}

		var score string
		switch mode {
		case "def-adj":
			score = colorNum(fmt.Sprintf("%.1f", r.AdjDefense))
		case "score":
			score = colorNum(fmt.Sprintf("%.2f", r.Score))
		default:
			score = colorNum(fmt.Sprintf("%d", r.Defense))
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			r.Name,
			r.Slot,
			cursed,
			colorNum(fmt.Sprintf("%d", r.BuffCount)),
			colorNum(fmt.Sprintf("%d", r.StatBonus)),
			score,
		})
		formatting = append(formatting, []string{
			`<ansi fg="yellow">%s</ansi>`,
			`<ansi fg="itemname">%s</ansi>`,
			`<ansi fg="white">%s</ansi>`,
			`%s`,
			`%s`, `%s`, `%s`,
		})
	}

	tblData := templates.GetTable(fmt.Sprintf("Armor Rankings by %s", titleSuffix), headers, rows, formatting...)
	tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}
