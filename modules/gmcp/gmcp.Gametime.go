package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	g := GMCPGametimeModule{
		plug: plugins.New(`gmcp.Gametime`, `1.0`),
	}

	events.RegisterListener(events.NewRound{}, g.newRoundHandler)

}

type GMCPGametimeModule struct {
	plug *plugins.Plugin
}

func (g *GMCPGametimeModule) newRoundHandler(e events.Event) events.ListenerReturn {

	_, typeOk := e.(events.NewRound)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "NewRound", "Actual Type", e.Type())
		return events.Cancel
	}

	gd := gametime.GetDate()

	payload := GMCPGametimeModule_Payload{
		Calendar:   gd.Calendar + "/" + configs.GetServerConfig().MudName.String(),
		Hour:       gd.Hour,
		Hour24:     gd.Hour24,
		Minute:     gd.Minute,
		AmPm:       gd.AmPm,
		Day:        gd.Day,
		Month:      gd.Month,
		MonthName:  gametime.MonthName(gd.Month),
		Year:       gd.Year,
		Zodiac:     gametime.GetZodiac(gd.Year),
		Night:      gd.Night,
		DayStart:   gd.DayStart,
		NightStart: gd.NightStart,
		SunCount:   gd.SunCount,
		MoonCount:  gd.MoonCount,
	}

	for _, user := range users.GetAllActiveUsers() {
		if !isGMCPEnabled(user.ConnectionId()) {
			continue
		}
		events.AddToQueue(GMCPOut{
			UserId:  user.UserId,
			Module:  `Gametime`,
			Payload: payload,
		})
	}

	return events.Continue
}

// /////////////////
// Gametime payload
// /////////////////
type GMCPGametimeModule_Payload struct {
	Calendar   string `json:"calendar"`
	Hour       int    `json:"hour"`
	Hour24     int    `json:"hour24"`
	Minute     int    `json:"minute"`
	AmPm       string `json:"ampm"`
	Day        int    `json:"day"`
	Month      int    `json:"month"`
	MonthName  string `json:"month_name"`
	Year       int    `json:"year"`
	Zodiac     string `json:"zodiac"`
	Night      bool   `json:"night"`
	DayStart   int    `json:"day_start"`
	NightStart int    `json:"night_start"`
	SunCount   int    `json:"sun_count"`  // expected to be 1 or 2
	MoonCount  int    `json:"moon_count"` // expected to be 0 - 3
}
