package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Killstats(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	otherSuggestions := []string{}

	tableTitle := `Kill Stats`

	var headers []string
	rows := [][]string{}

	formatting := []string{
		`<ansi fg="mobname">%s</ansi>`,
		`<ansi fg="red">%s</ansi>`,
		`<ansi fg="elite">%s</ansi>`,
		`<ansi fg="230">%s</ansi>`,
	}

	totalMobKills := 0
	totalPVPKills := 0
	//totalPVPDeaths := 0

	mobKills := map[string]int{}
	mobEliteKills := map[string]int{}
	raceKills := map[string]int{}
	areaKills := map[string]int{}
	charKills := map[string]int{}

	for mid, kCt := range user.Character.KD.Kills {

		if mobSpec := mobs.GetMobSpec(mobs.MobId(mid)); mobSpec != nil {

			totalMobKills += kCt

			// Populate mob kills
			mobKills[mobSpec.Character.Name] = mobKills[mobSpec.Character.Name] + kCt

			// Populate race kills
			if raceInfo := races.GetRace(mobSpec.Character.RaceId); raceInfo != nil {
				raceKills[raceInfo.Name] = raceKills[raceInfo.Name] + kCt
			}

			// Populate area kills
			areaKills[mobSpec.Zone] = areaKills[mobSpec.Zone] + kCt
		}
	}

	// Aggregate elite kills by mob name from "mobId:mobName" keys.
	for key, eCt := range user.Character.KD.EliteKills {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			mobEliteKills[parts[1]] = mobEliteKills[parts[1]] + eCt
		}
	}

	for userIdNameStr, killCount := range user.Character.KD.PlayerKills {
		parts := strings.Split(userIdNameStr, `:`)
		charKills[parts[1]] = killCount
		totalPVPKills++
	}

	renderStats := mobKills
	totalKills := totalMobKills
	totalDeaths := user.Character.KD.GetMobDeaths()
	kdRatio := user.Character.KD.GetMobKDRatio()

	if rest == `pvp` {

		renderStats = charKills
		totalKills = totalPVPKills
		kdRatio = user.Character.KD.GetPvpKDRatio()
		totalDeaths = user.Character.KD.GetPvpDeaths()

		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats area</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats race</ansi>`)

	} else if rest == `race` || rest == `races` {

		renderStats = raceKills
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats pvp</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats area</ansi>`)

	} else if rest == `zone` || rest == `zones` || rest == `area` || rest == `areas` {

		renderStats = areaKills
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats pvp</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats race</ansi>`)

	} else {

		rest = `mob` // default to mob

		renderStats = mobKills
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats pvp</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats area</ansi>`)
		otherSuggestions = append(otherSuggestions, `<ansi fg="command">killstats race</ansi>`)
	}

	isMobView := rest == `mob`

	if isMobView {
		headers = []string{strings.Title(rest), `Quantity`, `Elites`, `%`}
	} else {
		headers = []string{strings.Title(rest), `Quantity`, `%`}
	}

	for name, killCt := range renderStats {

		if isMobView {
			eliteCt := mobEliteKills[name]
			eliteStr := ``
			if eliteCt > 0 {
				eliteStr = fmt.Sprintf("%d", eliteCt)
			}
			rows = append(rows, []string{
				name,
				fmt.Sprintf("%d", killCt),
				eliteStr,
				fmt.Sprintf("%2.f%%", float64(killCt)/float64(totalKills)*100),
			})
		} else {
			rows = append(rows, []string{
				name,
				fmt.Sprintf("%d", killCt),
				fmt.Sprintf("%2.f%%", float64(killCt)/float64(totalKills)*100),
			})
		}
	}

	if isMobView {
		rows = append(rows, []string{``, ``, ``, ``})
		rows = append(rows, []string{`Total Kills`, fmt.Sprintf("%d", totalKills), ``, ``})
	} else {
		rows = append(rows, []string{``, ``, ``})
		rows = append(rows, []string{`Total Kills`, fmt.Sprintf("%d", totalKills), ``})
	}

	if totalDeaths == 0 {
		if isMobView {
			rows = append(rows, []string{`Total Deaths`, fmt.Sprintf("%d", totalDeaths), ``, `N/A`})
		} else {
			rows = append(rows, []string{`Total Deaths`, fmt.Sprintf("%d", totalDeaths), `N/A`})
		}
	} else {
		if isMobView {
			rows = append(rows, []string{`Total Deaths`, fmt.Sprintf("%d", totalDeaths), ``, fmt.Sprintf("%.2f:1", kdRatio)})
		} else {
			rows = append(rows, []string{`Total Deaths`, fmt.Sprintf("%d", totalDeaths), fmt.Sprintf("%.2f:1", kdRatio)})
		}
	}

	searchResultsTable := templates.GetTable(tableTitle+` by `+strings.Title(rest), headers, rows, formatting)
	tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
	tplTxt += fmt.Sprintf("Also try: %s\n", strings.Join(otherSuggestions, `, `))
	user.SendText(tplTxt)

	//user.SendText(fmt.Sprintf(`Also try: %s`, strings.Join(otherSuggestions, `, `)))

	return true, nil
}
