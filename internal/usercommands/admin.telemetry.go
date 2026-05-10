package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/telemetry"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* telemetry			(Admin only)
 */
func Telemetry(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 || args[0] == "help" {
		user.SendText(`<ansi fg="white-bold">Telemetry Commands:</ansi>`)
		user.SendText(`  <ansi fg="command">telemetry list item_drops</ansi>  [itemId=N] [mobId=N] [zone=X] [date=YYYYMMDD] [dateFrom=YYYYMMDD] [dateTo=YYYYMMDD] [asc|desc]`)
		user.SendText(`  <ansi fg="command">telemetry list item_pickups</ansi> [itemId=N] [zone=X] [date=YYYYMMDD] [asc|desc]`)
		user.SendText(`  <ansi fg="command">telemetry list mob_kills</ansi>    [mobId=N] [zone=X] [date=YYYYMMDD] [asc|desc]`)
		user.SendText(`  <ansi fg="command">telemetry list player_deaths</ansi>[mobId=N] [zone=X] [date=YYYYMMDD] [asc|desc]`)
		user.SendText(`  <ansi fg="command">telemetry list item_purchases</ansi>[itemId=N] [mobId=N] [date=YYYYMMDD] [asc|desc]`)
		user.SendText(`  <ansi fg="command">telemetry clear</ansi> [category] [date=YYYYMMDD]`)
		user.SendText(`  <ansi fg="command">telemetry save</ansi>`)
		return true, nil
	}

	switch args[0] {

	case "save":
		if err := telemetry.Save(); err != nil {
			user.SendText(fmt.Sprintf(`<ansi fg="red">Error saving telemetry: %s</ansi>`, err.Error()))
		} else {
			user.SendText(`Telemetry data saved.`)
		}
		return true, nil

	case "clear":
		category := ""
		date := ""
		if len(args) >= 2 {
			category = normCategory(args[1])
		}
		for _, arg := range args[2:] {
			if strings.HasPrefix(arg, "date=") {
				date = strings.TrimPrefix(arg, "date=")
			}
		}
		telemetry.Clear(category, "", date, "", 0, 0, 0)
		user.SendText(`Telemetry records cleared.`)
		return true, nil

	case "list":
		if len(args) < 2 {
			user.SendText(`Usage: telemetry list <category> [filters...]`)
			return true, nil
		}

		category := normCategory(args[1])
		if category == "" {
			user.SendText(fmt.Sprintf(`Unknown category: %s`, args[1]))
			return true, nil
		}

		qb := telemetry.Query().Category(category)
		descSort := true

		for _, arg := range args[2:] {
			switch {
			case arg == "asc":
				descSort = false
			case arg == "desc":
				descSort = true
			case strings.HasPrefix(arg, "itemid="):
				if v, err := strconv.Atoi(strings.TrimPrefix(arg, "itemid=")); err == nil {
					qb = qb.ItemId(v)
				}
			case strings.HasPrefix(arg, "mobid="):
				if v, err := strconv.Atoi(strings.TrimPrefix(arg, "mobid=")); err == nil {
					qb = qb.MobId(v)
				}
			case strings.HasPrefix(arg, "roomid="):
				if v, err := strconv.Atoi(strings.TrimPrefix(arg, "roomid=")); err == nil {
					qb = qb.RoomId(v)
				}
			case strings.HasPrefix(arg, "zone="):
				qb = qb.Zone(strings.TrimPrefix(arg, "zone="))
			case strings.HasPrefix(arg, "date="):
				qb = qb.Date(strings.TrimPrefix(arg, "date="))
			case strings.HasPrefix(arg, "datefrom="):
				qb = qb.DateFrom(strings.TrimPrefix(arg, "datefrom="))
			case strings.HasPrefix(arg, "dateto="):
				qb = qb.DateTo(strings.TrimPrefix(arg, "dateto="))
			}
		}

		if descSort {
			qb = qb.SortDesc()
		} else {
			qb = qb.SortAsc()
		}

		results := qb.Results()
		if len(results) == 0 {
			user.SendText(`No telemetry records found.`)
			return true, nil
		}

		headers := []string{`Date`, `Category`, `ItemId`, `MobId`, `RoomId`, `Zone`, `Count`}
		formatting := []string{
			`<ansi fg="cyan">%s</ansi>`,
			`<ansi fg="white">%s</ansi>`,
			`<ansi fg="itemname">%s</ansi>`,
			`<ansi fg="mobname">%s</ansi>`,
			`%s`,
			`<ansi fg="room-title">%s</ansi>`,
			`<ansi fg="yellow-bold">%s</ansi>`,
		}

		rows := make([][]string, 0, len(results))
		for _, r := range results {
			rows = append(rows, []string{
				r.Date,
				r.Category,
				intOrEmpty(r.ItemId),
				intOrEmpty(r.MobId),
				intOrEmpty(r.RoomId),
				r.Zone,
				strconv.Itoa(r.Count),
			})
		}

		title := fmt.Sprintf(`Telemetry: %s (%d records)`, category, len(results))
		tblData := templates.GetTable(title, headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", tblData, user.UserId)
		user.SendText(tplTxt)

		return true, nil
	}

	user.SendText(fmt.Sprintf(`Unknown telemetry command: %s`, args[0]))
	return true, nil
}

func normCategory(s string) string {
	switch s {
	case "item_drops", "item_drop":
		return telemetry.CatItemDrop
	case "item_pickups", "item_pickup":
		return telemetry.CatItemPickup
	case "mob_kills", "mob_kill":
		return telemetry.CatMobKill
	case "player_deaths", "player_death":
		return telemetry.CatPlayerDeath
	case "item_purchases", "item_purchase":
		return telemetry.CatItemPurchase
	}
	return ""
}

func intOrEmpty(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}
