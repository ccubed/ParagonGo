package gametime

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const roundDateCacheMax = 20

var (
	dayResetOffset int = 0

	roundDateCache    = map[uint64]GameDate{}
	roundDateCacheSeq []uint64
)

// anchorPeriods maps the user-facing token (already lowercased and trimmed of any
// "x" prefix down to its first three characters by AddPeriod) to the canonical
// period name understood by GameDate.LastPeriod.
//
// These are "anchor" periods: rather than advancing by a fixed duration, they
// first snap backwards to the most recent occurrence of a named moment (e.g. the
// last noon, the last sunrise) and then advance forward by whole days. This is why
// AddPeriod treats them differently from plain durations like "3 days".
//
// Note that "midnight" intentionally maps to the "day" anchor: midnight *is* the
// start of a game day, so the last midnight is the last day-start.
var anchorPeriods = map[string]string{
	`noo`:      `noon`,    // "noon", "noons"
	`mid`:      `day`,     // "midnight", "midnights" -> start of day
	`sunrise`:  `sunrise`, // matched in full because "sun" alone is ambiguous
	`sunrises`: `sunrise`,
	`sunset`:   `sunset`,
	`sunsets`:  `sunset`,
}

type RoundTimer struct {
	RoundStart uint64 `yaml:"roundstart,omitempty"`
	Period     string `yaml:"period,omitempty"`
	gd         GameDate
}

func (r *RoundTimer) Expired() bool {
	if r.Period == `` || r.RoundStart == 0 {
		return true
	}
	if r.gd.RoundNumber == 0 {
		r.gd = GetDate(r.RoundStart)
	}
	return r.gd.AddPeriod(r.Period) < util.GetRoundCount()
}

type GameDate struct {
	// which calendar this gamedate uses
	Calendar string
	// The round number this GameDate represents
	RoundNumber      uint64
	RoundsPerDay     int
	NightHoursPerDay int

	Year        int
	Month       int
	Week        int
	Day         int
	Hour        int
	Hour24      int
	Minute      int
	MinuteFloat float64
	AmPm        string
	Night       bool

	DayStart   int
	NightStart int
	DuskHours  int
	SunCount   int
	MoonCount  int
}

func (gd GameDate) String(symbolOnly ...bool) string {

	dayNight := `day`
	if gd.Night {
		dayNight = `night`
	} else {
		if gd.NightStart-gd.Hour24 < gd.DuskHours {
			dayNight = `day-dusk`
		}
	}

	if len(symbolOnly) > 0 && symbolOnly[0] {

		if gd.Night {
			return `<ansi fg="night">☾</ansi>` // •
		}
		return fmt.Sprintf(`<ansi fg="%s">☀️</ansi>`, dayNight) //
	}

	return fmt.Sprintf("<ansi fg=\"%s\">%d:%02d%s</ansi>", dayNight, gd.Hour, gd.Minute, gd.AmPm)
}

// SetToNight jumps the global clock forward to the next night.
//
// If a roundAdjustment is provided it is added to (or subtracted from) the target
// round. This is useful to land on the round right before the rollover.
func SetToNight(roundAdjustment ...int) {
	setToDayPart(`sunset`, roundAdjustment...)
}

// SetToDay jumps the global clock forward to the next day.
//
// If a roundAdjustment is provided it is added to (or subtracted from) the target
// round. This is useful to land on the round right before the rollover.
func SetToDay(roundAdjustment ...int) {
	setToDayPart(`sunrise`, roundAdjustment...)
}

// setToDayPart is the shared implementation behind SetToNight/SetToDay. Both
// functions are identical except for the anchor they snap to (sunset vs sunrise),
// so the logic lives here once:
//   - find the last occurrence of the anchor,
//   - apply the optional round adjustment,
//   - advance one day so we land on the *next* occurrence,
//   - write the result back to the global round counter.
func setToDayPart(anchor string, roundAdjustment ...int) {

	dayRound := GetLastPeriod(anchor, util.GetRoundCount())

	if len(roundAdjustment) > 0 {
		if roundAdjustment[0] < 0 {
			dayRound -= uint64(-1 * roundAdjustment[0])
		} else {
			dayRound += uint64(roundAdjustment[0])
		}
	}

	gd := GetDate(dayRound).Add(0, 1, 0)
	util.SetRoundCount(gd.RoundNumber)
}

// SetTime jumps the global clock forward to a specific hour (0-23) and optional
// minute. It works by adjusting dayResetOffset so the next ReCalculate reports the
// requested time, then clears the round-date cache so stale entries are not served.
func SetTime(setToHour int, setToMinutes ...int) {

	rpd := activeCalendar[`default`].roundsPerDay
	rph := activeCalendar[`default`].roundsPerHour

	setToHour = setToHour % 24
	dayResetOffset = int(math.Floor(float64(setToHour) * rph))
	if len(setToMinutes) > 0 {
		dayResetOffset += int(math.Ceil((float64(setToMinutes[0]) / 60) * rph))
	}

	roundOfDay := int(util.GetRoundCount() % rpd)
	dayResetOffset -= roundOfDay

	// Reset the cache
	clear(roundDateCache)
	roundDateCacheSeq = roundDateCacheSeq[:0]
}

func IsNight() bool {
	gd := GetDate()
	return gd.Night
}

// Gets the details of the current date
func GetDate(forceRound ...uint64) GameDate {

	currentRound := uint64(0)
	if len(forceRound) > 0 {
		currentRound = forceRound[0]
	} else {
		currentRound = util.GetRoundCount()
	}

	if d, ok := roundDateCache[currentRound]; ok {
		return d
	}

	if len(roundDateCache) >= roundDateCacheMax {
		delete(roundDateCache, roundDateCacheSeq[0])
		roundDateCacheSeq = roundDateCacheSeq[1:]
	}

	roundDateCache[currentRound] = getDate(currentRound)
	roundDateCacheSeq = append(roundDateCacheSeq, currentRound)

	return roundDateCache[currentRound]
}

func getDate(currentRound uint64) GameDate {
	return getDateForCalendar(currentRound, `default`)
}

// getDateForCalendar builds a fully-populated GameDate for a round under a
// specific named calendar. getDate is the common "default" calendar case; the
// anchor path in AddPeriod uses this directly with g.Calendar so that day-stepping
// after a snap (e.g. "1 sunrise") respects the originating date's calendar rather
// than silently reverting to "default".
func getDateForCalendar(currentRound uint64, calendarToUse string) GameDate {

	gd := GameDate{Calendar: calendarToUse}

	ac := activeCalendar[calendarToUse]

	gd.RoundNumber = currentRound
	gd.RoundsPerDay = int(ac.roundsPerDay)
	gd.NightHoursPerDay = ac.nightHours
	gd.DuskHours = ac.duskHours
	gd.SunCount = ac.sunCount
	gd.MoonCount = ac.moonCount

	gd.ReCalculate()

	return gd
}

func (g *GameDate) ReCalculate() {

	ac := activeCalendar[g.Calendar]

	currentRoundAdjusted := (g.RoundNumber + uint64(dayResetOffset))
	roundOfDay := int(currentRoundAdjusted % uint64(g.RoundsPerDay))

	hourFloat, minutesFloat := math.Modf(float64(roundOfDay) / float64(g.RoundsPerDay) * 24)

	hour := int(hourFloat)
	hour24 := hour

	night := false
	halfNight := int(math.Floor(float64(g.NightHoursPerDay) / 2))
	nightStart := 24 - halfNight
	nightEnd := int(g.NightHoursPerDay) - halfNight
	if hour >= nightStart || hour < nightEnd {
		night = true
	}

	ampm := `AM`
	if hour >= 12 {
		ampm = `PM`
		hour -= 12
	}

	if hour == 0 {
		hour = 12
	}

	minute := math.Floor(minutesFloat * 60)

	daysPerYear := float64(ac.daysPerYear)
	numMonths := ac.numMonths

	day := math.Floor(float64(currentRoundAdjusted)/float64(g.RoundsPerDay)) + 1
	year := math.Ceil(day / daysPerYear)

	if year > 1 {
		day -= math.Floor((year - 1) * daysPerYear)
	}
	week := math.Floor(float64(day) / float64(ac.daysPerWeek))

	month := 1 + math.Floor((day*24)/ac.hoursPerMonth)
	if int(month) > numMonths {
		month = float64(numMonths)
	}

	g.Day = int(day)
	g.Year = int(year)
	g.Month = int(month)
	g.Week = int(week)
	g.Hour = hour
	g.Hour24 = hour24
	g.Minute = int(minute)
	g.MinuteFloat = minutesFloat * 60
	g.AmPm = ampm
	g.Night = night

	g.NightStart = nightStart
	g.DayStart = nightEnd
}

func (g GameDate) Add(adjustHours int, adjustDays int, adjustYears int) GameDate {

	ac := activeCalendar[g.Calendar]
	rStart := g.RoundNumber

	if adjustYears != 0 {
		yearRounds := uint64(ac.daysPerYear) * uint64(g.RoundsPerDay)
		if adjustYears < 1 {
			g.RoundNumber -= uint64(-1*adjustYears) * yearRounds
		} else {
			g.RoundNumber += uint64(adjustYears) * yearRounds
		}
	}

	if adjustDays != 0 {
		if adjustDays < 1 {
			g.RoundNumber -= uint64(-1 * adjustDays * g.RoundsPerDay)
		} else {
			g.RoundNumber += uint64(adjustDays * g.RoundsPerDay)
		}
	}

	if adjustHours != 0 {
		rph := ac.roundsPerHour
		if adjustHours < 1 {
			g.RoundNumber -= uint64(math.Floor(-1 * float64(adjustHours) * rph))
		} else {
			g.RoundNumber += uint64(math.Floor(float64(adjustHours) * rph))
		}
	}

	if rStart != g.RoundNumber {
		g.ReCalculate()
	}

	return g
}

// realTimeRounds converts a real-world quantity into a number of game rounds,
// based on the configured real seconds per round. It is used by AddPeriod when a
// period string contains "real" or "irl".
//
// Note: a "real day" is treated as 84600 seconds (23.5 hours), not 86400. This is
// a long-standing intentional quirk of this engine, preserved here so existing
// content that relies on it keeps the same timing.
type realTimeRounds struct {
	perMinute int
	perHour   int
	perDay    int
}

func newRealTimeRounds() realTimeRounds {
	// RoundSeconds is a real-time value — read from the timing config, not the calendar.
	roundSeconds := int(configs.GetTimingConfig().RoundSeconds)
	if roundSeconds < 1 {
		roundSeconds = 1
	}
	return realTimeRounds{
		perMinute: 60 / roundSeconds,
		perHour:   3600 / roundSeconds,
		perDay:    84600 / roundSeconds,
	}
}

// AddPeriod returns the round number reached by advancing FORWARD from this
// GameDate by the supplied period string. It is the additive counterpart to
// StartOf (which snaps backward).
//
// Example:
//
//	gd := gametime.GetDate()
//	nextPeriodRound := gd.AddPeriod(`10 days`)
//
// Accepts: x years, x months, x weeks, x days, x hours, x minutes, x rounds, and
// the day-anchor units noon/midnight/sunrise/sunset (which snap to the last such
// moment and then add x days).
//
// If `IRL` or `real` appear in the string, such as `x irl days` or `x days irl`,
// real-world time is used instead of game time. Real time is not supported for the
// day-anchor units; specifying it logs an error and falls back to game time.
func (g GameDate) AddPeriod(periodStr string) uint64 {

	qty, timeStr, realTime := parsePeriod(periodStr)
	if timeStr == `` && qty == 0 {
		// Empty / unparseable-as-anything input: no movement.
		return g.RoundNumber
	}

	if len(timeStr) >= 3 {

		strShort := timeStr[0:3]

		// --- Day-anchor units (noon/midnight/sunrise/sunset) ---------------------
		// These do not advance by a fixed duration. They first snap backward to the
		// last occurrence of the anchor, then advance forward by qty whole days.
		// Handled first, and via a shared table, so the four near-identical branches
		// that previously existed are collapsed into one.
		anchorKey := strShort
		if anchorKey != `noo` && anchorKey != `mid` {
			// sunrise/sunset are matched in full (their first three letters, "sun",
			// are ambiguous), so use the whole token for the lookup.
			anchorKey = timeStr
		}
		if anchor, ok := anchorPeriods[anchorKey]; ok {
			if realTime {
				mudlog.Error("AddPeriod", "error", "real time not supported for "+timeStr)
			}
			anchored := getDateForCalendar(g.LastPeriod(anchor), g.Calendar)
			return anchored.Add(0, qty, 0).RoundNumber
		}

		// --- Fixed-duration units -----------------------------------------------
		switch strShort {

		case `yea`: // year / years / yearly
			if realTime {
				rt := newRealTimeRounds()
				return g.RoundNumber + uint64(qty*rt.perDay*365)
			}
			return g.Add(0, 0, qty).RoundNumber

		case `mon`: // month / months / monthly
			if realTime {
				rt := newRealTimeRounds()
				return g.RoundNumber + uint64(qty*rt.perHour*730)
			}
			hoursPerMonth := activeCalendar[g.Calendar].hoursPerMonth
			return g.Add(int(math.Round(hoursPerMonth))*qty, 0, 0).RoundNumber

		case `wee`: // week / weeks / weekly
			if realTime {
				rt := newRealTimeRounds()
				return g.RoundNumber + uint64(qty*rt.perDay*7)
			}
			return g.Add(0, activeCalendar[g.Calendar].daysPerWeek*qty, 0).RoundNumber

		case `day`, `dai`: // day / days / daily
			if realTime {
				rt := newRealTimeRounds()
				return g.RoundNumber + uint64(qty*rt.perDay)
			}
			return g.Add(0, qty, 0).RoundNumber

		case `hou`: // hour / hours / hourly
			if realTime {
				rt := newRealTimeRounds()
				return g.RoundNumber + uint64(qty*rt.perHour)
			}
			return g.Add(qty, 0, 0).RoundNumber

		case `min`: // minute / minutes
			if realTime {
				rt := newRealTimeRounds()
				return g.RoundNumber + uint64(qty*rt.perMinute)
			}
			return g.RoundNumber + uint64(math.Floor(float64(qty)*activeCalendar[g.Calendar].roundsPerMinute))
		}

		// Unrecognised unit: fail over to treating qty as raw rounds.
		return g.RoundNumber + uint64(qty)
	}

	// No unit string at all (e.g. just a number): treat qty as hours, matching the
	// historical default behaviour.
	return g.Add(qty, 0, 0).RoundNumber
}

// parsePeriod breaks a period string such as "2 days", "3 irl hours" or "5" into
// its component quantity, unit token and a real-time flag.
//
// Supported shapes:
//
//	"<unit>"                e.g. "day"            -> qty 1
//	"<qty>"                 e.g. "5"              -> raw rounds (no unit)
//	"<qty> <unit>"          e.g. "2 days"
//	"<qty> <real> <unit>"   e.g. "2 irl days"
//	"<qty> <unit> <real>"   e.g. "2 days irl"
//	"<qty> <game> <unit>"   e.g. "2 game days"    (explicit game time)
//	"<qty> <unit> <game>"   e.g. "2 days gametime"
//
// qty defaults to 1 whenever it is missing or not a positive integer.
func parsePeriod(periodStr string) (qty int, timeStr string, realTime bool) {

	qty = 1

	if periodStr == `` {
		return 0, ``, false
	}

	parts := strings.Split(strings.ToLower(periodStr), ` `)

	switch len(parts) {

	case 1: // either a bare number, or a bare unit
		// Try to parse a number; if that fails (or is < 1) it must be a unit string.
		if n, err := strconv.Atoi(parts[0]); err == nil && n >= 1 {
			qty = n
		} else {
			timeStr = parts[0]
		}

	case 2: // "<qty> <unit>"
		if n, _ := strconv.Atoi(parts[0]); n >= 1 {
			qty = n
		}
		timeStr = parts[1]

	case 3: // "<qty> <qualifier> <unit>" or "<qty> <unit> <qualifier>"
		if n, _ := strconv.Atoi(parts[0]); n >= 1 {
			qty = n
		}
		switch {
		case parts[1] == `real` || parts[1] == `irl`:
			realTime = true
			timeStr = parts[2]
		case parts[1] == `game` || parts[1] == `gametime`:
			timeStr = parts[2]
		case parts[2] == `real` || parts[2] == `irl`:
			realTime = true
			timeStr = parts[1]
		case parts[2] == `game` || parts[2] == `gametime`:
			timeStr = parts[1]
		}
	}

	return qty, timeStr, realTime
}

// StartOf snaps BACKWARD to the start of the named period relative to this
// GameDate, returning that round number. It is the backward-snapping counterpart
// to AddPeriod and never returns a round later than g.RoundNumber.
//
// It accepts both bare period names ("hour", "day", "week", "month", "year",
// "noon", "sunrise", "sunset") and the natural-language "start of X" forms used by
// player/scripting input. The connective words "start", "of" and "the" are
// stripped during normalisation, so all of these are equivalent:
//
//	"start of the hour"   "start of hour"   "start hour"   "hour"
//
// IMPORTANT: this is a backward operation. Callers that expect a future expiry
// round (the common AddPeriod use case) must NOT route "start of X" strings here
// expecting forward movement — the result will be in the past or present.
func (g GameDate) StartOf(periodStr string) uint64 {
	name := normalizeStartOf(periodStr)
	if name == `` {
		// Nothing recognisable to snap to; stay put.
		return g.RoundNumber
	}
	return g.LastPeriod(name)
}

// normalizeStartOf lowercases the input and removes the connective tokens
// "start", "of" and "the", leaving just the period name. e.g. "Start Of The Day"
// becomes "day". Returns "" if nothing is left.
func normalizeStartOf(periodStr string) string {
	parts := strings.Fields(strings.ToLower(periodStr))
	kept := parts[:0]
	for _, p := range parts {
		switch p {
		case `start`, `of`, `the`:
			// drop connective words
		default:
			kept = append(kept, p)
		}
	}
	if len(kept) == 0 {
		return ``
	}
	// Only the period name is meaningful; ignore any trailing words.
	return kept[0]
}

// LastPeriod returns the round number at which the named period last began,
// relative to this GameDate's round number, honouring this GameDate's calendar.
//
// This is the calendar-aware core used by both StartOf and AddPeriod's day-anchor
// handling. Previously this logic lived only in the package-level GetLastPeriod,
// which always used the "default" calendar; making it a method ensures a non-
// default calendar's AddPeriod("1 sunrise") and StartOf calls compute against the
// correct calendar.
//
// Supported names: hour, day (== midnight), week, month, year, noon, sunrise,
// sunset. Unknown names return the round number unchanged.
func (g GameDate) LastPeriod(periodName string) uint64 {

	ac := activeCalendar[g.Calendar]
	roundNumber := g.RoundNumber

	roundsPerDay := ac.roundsPerDay
	nightHoursPerDay := uint64(ac.nightHours)
	roundsPerHour := ac.roundsPerHour
	noonRound := ac.noonRound

	// Offsets of the current round within each enclosing period.
	roundOfWeek := roundNumber % ac.roundsPerWeek
	roundOfDay := roundNumber % roundsPerDay                      // since midnight
	roundOfHour := roundOfDay % uint64(math.Floor(roundsPerHour)) // since top of hour

	switch periodName {

	case `hour`: // start of the current hour
		roundNumber -= roundOfHour

	case `day`, `midnight`: // start of the current day (midnight)
		roundNumber -= roundOfDay

	case `week`: // start of the current week
		roundNumber -= roundOfWeek

	case `month`: // start of the current month
		// Walk back to the most recent round whose day is the first of the month.
		// hoursPerMonth is fractional in general, so there is no closed-form round
		// offset; instead snap to midnight and step whole days backward until the
		// month rolls over. Bounded by the days in a month, so it is cheap.
		roundNumber -= roundOfDay // first go to midnight today
		startMonth := getDateForCalendar(roundNumber, g.Calendar).Month
		for roundNumber >= roundsPerDay {
			prev := roundNumber - roundsPerDay
			if getDateForCalendar(prev, g.Calendar).Month != startMonth {
				break
			}
			roundNumber = prev
		}

	case `year`: // start of the current year
		// day is 1-based within the year, so (day-1) whole days have elapsed.
		roundNumber -= roundOfDay // midnight today
		dayOfYear := uint64(getDateForCalendar(roundNumber, g.Calendar).Day)
		if dayOfYear > 1 {
			roundNumber -= (dayOfYear - 1) * roundsPerDay
		}

	case `noon`: // last time 12pm was reached
		roundNumber -= roundOfDay
		if roundOfDay < noonRound {
			// Noon has not happened yet today; the last noon was yesterday.
			roundNumber -= roundsPerDay - noonRound
		} else {
			// Noon already passed today.
			roundNumber += noonRound
		}

	case `sunrise`: // last sunrise (start of day + half the night)
		roundNumber -= roundOfDay                                                       // strip today's rounds
		roundNumber -= roundsPerDay                                                     // back up a day
		roundNumber += uint64(math.Ceil(float64(nightHoursPerDay) / 2 * roundsPerHour)) // add half a night

	case `sunset`: // last sunset (next midnight minus half the night)
		roundNumber -= roundOfDay                                                       // strip today's rounds
		roundNumber -= uint64(math.Ceil(float64(nightHoursPerDay) / 2 * roundsPerHour)) // subtract half a night
	}

	return roundNumber
}

// GetLastPeriod is a package-level convenience wrapper around GameDate.LastPeriod
// for the "default" calendar. It is retained for callers that only have a round
// number on hand (e.g. SetToNight/SetToDay) and do not need calendar selection.
//
// Prefer GameDate.LastPeriod when you already hold a GameDate, as it respects that
// date's calendar.
func GetLastPeriod(periodName string, roundNumber uint64) uint64 {
	g := GameDate{Calendar: `default`, RoundNumber: roundNumber}
	return g.LastPeriod(periodName)
}

func MonthName(month int) string {
	names := activeCalendar[`default`].monthNames
	if len(names) == 0 {
		return ``
	}
	month--
	return names[month%len(names)]
}

func GetZodiac(year int) string {
	z := activeCalendar[`default`].zodiacList
	if len(z) == 0 {
		return ``
	}
	return z[year%len(z)]
}
