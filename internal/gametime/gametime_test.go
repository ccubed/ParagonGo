package gametime

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// seedCalendarConfig applies a CalendarConfig under "default" for the duration
// of the test, restoring the previous activeCalendar map on cleanup.
func seedCalendarConfig(t *testing.T, c CalendarConfig) {
	t.Helper()
	prev := activeCalendar
	applyCalendarConfig(`default`, c)
	t.Cleanup(func() {
		activeCalendar = prev
		clear(roundDateCache)
		roundDateCacheSeq = roundDateCacheSeq[:0]
	})
}

// defaultTestCalendar returns a CalendarConfig suitable for most tests:
// 240 rounds/day, 8 night hours, 3 dusk hours, 365 days/year, 7 days/week, 12 months.
func defaultTestCalendar(roundsPerDay, nightHours int) CalendarConfig {
	return CalendarConfig{
		RoundsPerDay: roundsPerDay,
		NightHours:   nightHours,
		DuskHours:    3,
		DaysPerYear:  365,
		DaysPerWeek:  7,
		Months:       []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
		Zodiac:       []string{"Rat"},
	}
}

// newGD constructs a GameDate with the given fields and calls ReCalculate.
// It seeds activeCalendar["default"] to match so that ReCalculate uses the right derived constants.
func newGD(roundNumber uint64, roundsPerDay, nightHoursPerDay int) GameDate {
	applyCalendarConfig(`default`, defaultTestCalendar(roundsPerDay, nightHoursPerDay))
	gd := GameDate{
		Calendar:         `default`,
		RoundNumber:      roundNumber,
		RoundsPerDay:     roundsPerDay,
		NightHoursPerDay: nightHoursPerDay,
		DuskHours:        3,
	}
	gd.ReCalculate()
	return gd
}

// --- ReCalculate ---

func Test_ReCalculate_HourAndMinute(t *testing.T) {
	// 240 rounds/day => 10 rounds/hour.
	// Round 0 => hour 0 (midnight), round 10 => hour 1, round 120 => noon.
	rpd := 240

	cases := []struct {
		round      uint64
		wantHour24 int
		wantMinute int
		wantAmPm   string
	}{
		{0, 0, 0, "AM"},
		{10, 1, 0, "AM"},
		{120, 12, 0, "PM"},
		{125, 12, 30, "PM"},
		{230, 23, 0, "PM"},
	}

	for _, tc := range cases {
		gd := newGD(tc.round, rpd, 0)
		if gd.Hour24 != tc.wantHour24 {
			t.Errorf("round %d: Hour24 = %d, want %d", tc.round, gd.Hour24, tc.wantHour24)
		}
		if gd.Minute != tc.wantMinute {
			t.Errorf("round %d: Minute = %d, want %d", tc.round, gd.Minute, tc.wantMinute)
		}
		if gd.AmPm != tc.wantAmPm {
			t.Errorf("round %d: AmPm = %q, want %q", tc.round, gd.AmPm, tc.wantAmPm)
		}
	}
}

func Test_ReCalculate_NightFlag(t *testing.T) {
	// 240 rounds/day, 8 night hours => nightStart=20, nightEnd=4.
	// Night: hour >= 20 or hour < 4.
	rpd := 240
	nightHours := 8

	cases := []struct {
		hour24    int
		wantNight bool
	}{
		{0, true},
		{3, true},
		{4, false},
		{12, false},
		{19, false},
		{20, true},
		{23, true},
	}

	for _, tc := range cases {
		round := uint64(tc.hour24 * (rpd / 24))
		gd := newGD(round, rpd, nightHours)
		if gd.Night != tc.wantNight {
			t.Errorf("hour24=%d: Night=%v, want %v", tc.hour24, gd.Night, tc.wantNight)
		}
	}
}

func Test_ReCalculate_MonthNeverExceedsNumMonths(t *testing.T) {
	// Walk every day of a year and confirm Month stays in [1, numMonths].
	rpd := 240
	for dayOfYear := 1; dayOfYear <= 366; dayOfYear++ {
		round := uint64((dayOfYear - 1) * rpd)
		gd := newGD(round, rpd, 0)
		if gd.Month < 1 || gd.Month > activeCalendar[`default`].numMonths {
			t.Errorf("day %d: Month = %d, want 1-%d", dayOfYear, gd.Month, activeCalendar[`default`].numMonths)
		}
	}
}

func Test_ReCalculate_YearAndDayRollover(t *testing.T) {
	rpd := 240
	// Day 365 of year 1 => last day of year.
	lastDayRound := uint64(364 * rpd)
	gd := newGD(lastDayRound, rpd, 0)
	if gd.Year != 1 {
		t.Errorf("year at day 365: got %d, want 1", gd.Year)
	}
	if gd.Day != 365 {
		t.Errorf("day at end of year: got %d, want 365", gd.Day)
	}

	// First round of year 2.
	firstRoundY2 := uint64(365 * rpd)
	gd2 := newGD(firstRoundY2, rpd, 0)
	if gd2.Year != 2 {
		t.Errorf("year at start of year 2: got %d, want 2", gd2.Year)
	}
	if gd2.Day != 1 {
		t.Errorf("day at start of year 2: got %d, want 1", gd2.Day)
	}
}

// --- String / dusk detection ---

func Test_String_DuskOnlyApproachingNight(t *testing.T) {
	// 240 rounds/day, 0 night hours => nightStart=24 (no real night).
	// With nightStart=24, NightStart-Hour24 is always >= 0 for hour24 in [0,23].
	// Dusk fires when NightStart-Hour24 < DuskHours (3), i.e. hour24 >= 22.
	rpd := 240

	cases := []struct {
		hour24   int
		wantDusk bool
	}{
		{21, false},
		{22, true},
		{23, true},
	}

	for _, tc := range cases {
		round := uint64(tc.hour24 * (rpd / 24))
		gd := newGD(round, rpd, 0)
		s := gd.String()
		hasDusk := containsSubstr(s, "day-dusk")
		if hasDusk != tc.wantDusk {
			t.Errorf("hour24=%d: dusk in String=%v, want %v (got %q)",
				tc.hour24, hasDusk, tc.wantDusk, s)
		}
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

// --- GetDate cache ---

func Test_GetDate_CacheEviction(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// Reset global cache state.
	clear(roundDateCache)
	roundDateCacheSeq = roundDateCacheSeq[:0]

	// Populate exactly roundDateCacheMax entries.
	for i := uint64(1000); i < uint64(1000+roundDateCacheMax); i++ {
		GetDate(i)
	}
	if len(roundDateCache) != roundDateCacheMax {
		t.Fatalf("after filling: cache size = %d, want %d", len(roundDateCache), roundDateCacheMax)
	}

	// Adding one more entry must evict the oldest (round 1000).
	GetDate(uint64(1000 + roundDateCacheMax))
	if len(roundDateCache) != roundDateCacheMax {
		t.Errorf("after eviction: cache size = %d, want %d", len(roundDateCache), roundDateCacheMax)
	}
	if _, ok := roundDateCache[1000]; ok {
		t.Error("oldest entry (1000) should have been evicted but is still present")
	}
	if _, ok := roundDateCache[uint64(1000+roundDateCacheMax)]; !ok {
		t.Error("newest entry should be present after eviction")
	}
}

func Test_GetDate_CacheHit(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	clear(roundDateCache)
	roundDateCacheSeq = roundDateCacheSeq[:0]

	const r = uint64(42000)
	first := GetDate(r)
	// Mutate the cached copy to detect whether a new computation is returned.
	roundDateCache[r] = GameDate{RoundNumber: 99999}
	second := GetDate(r)
	if second.RoundNumber != 99999 {
		t.Error("GetDate should return cached value on second call")
	}
	_ = first
}

// --- GetLastPeriod ---

func Test_GetLastPeriod_Day(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// Round 250 = day 2, round 10 into the day.
	// Last midnight = round 240.
	got := GetLastPeriod("day", 250)
	if got != 240 {
		t.Errorf("GetLastPeriod(day, 250): got %d, want 240", got)
	}
}

func Test_GetLastPeriod_Hour(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// 240 rounds/day => 10 rounds/hour.
	// Round 255 = 15 rounds into hour 1 of day 2 => last hour start = 250.
	got := GetLastPeriod("hour", 255)
	if got != 250 {
		t.Errorf("GetLastPeriod(hour, 255): got %d, want 250", got)
	}
}

func Test_GetLastPeriod_Noon_AfterNoon(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// 240 rounds/day => noon at round 120 of each day.
	// Round 130 = 10 rounds past noon on day 1 => last noon = 120.
	got := GetLastPeriod("noon", 130)
	if got != 120 {
		t.Errorf("GetLastPeriod(noon, 130): got %d, want 120", got)
	}
}

func Test_GetLastPeriod_Noon_BeforeNoon(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// Round 50 = before noon on day 1 => last noon was yesterday (day 0, round -120).
	// Day 0 doesn't exist (round 0 is midnight day 1), so last noon = round 0 + 120 - 240 = -120.
	// In uint64 arithmetic: roundNumber(50) - roundOfDay(50) - (240 - 120) = 50 - 50 - 120 = -120.
	// This underflows; the function returns that underflowed value. Verify the formula is correct
	// by checking the relative offset rather than the absolute value.
	got := GetLastPeriod("noon", 50)
	// The result should be exactly 120 rounds before midnight of the current day.
	// midnight of day containing round 50 = 50 - (50%240) = 0.
	// noon of previous day = 0 - (240 - 120) = -120 (wraps in uint64).
	wantMidnight := uint64(50) - uint64(50%240)      // = 0
	noonRound := uint64(240 / 2)                     // = 120
	want := wantMidnight - (uint64(240) - noonRound) // = 0 - 120 (uint64 wrap)
	if got != want {
		t.Errorf("GetLastPeriod(noon, 50): got %d, want %d", got, want)
	}
}

func Test_GetLastPeriod_Noon_ExactlyAtNoon(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// Round 120 is exactly noon on day 1.
	// roundOfDay = 120, noonRound = 120; roundOfDay >= noonRound => today's noon branch.
	// result = 0 + 120 = 120.
	got := GetLastPeriod("noon", 120)
	if got != 120 {
		t.Errorf("GetLastPeriod(noon, 120): got %d, want 120", got)
	}
}

func Test_GetLastPeriod_Week(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// 7 days = 1680 rounds. Round 1700 = 20 rounds into week 2.
	// Last week start = 1700 - (1700 % 1680) = 1700 - 20 = 1680.
	got := GetLastPeriod("week", 1700)
	if got != 1680 {
		t.Errorf("GetLastPeriod(week, 1700): got %d, want 1680", got)
	}
}

// --- AddPeriod ---

func Test_AddPeriod_Days(t *testing.T) {
	// Use GameDate directly; AddPeriod for "days" calls g.Add which uses g.RoundsPerDay.
	gd := newGD(0, 240, 0)
	got := gd.AddPeriod("3 days")
	want := uint64(3 * 240)
	if got != want {
		t.Errorf("AddPeriod(3 days): got %d, want %d", got, want)
	}
}

func Test_AddPeriod_Hours(t *testing.T) {
	gd := newGD(0, 240, 0)
	got := gd.AddPeriod("2 hours")
	want := uint64(2 * (240 / 24)) // 2 * 10 = 20
	if got != want {
		t.Errorf("AddPeriod(2 hours): got %d, want %d", got, want)
	}
}

func Test_AddPeriod_Weeks(t *testing.T) {
	gd := newGD(0, 240, 0)
	got := gd.AddPeriod("1 week")
	want := uint64(7 * 240)
	if got != want {
		t.Errorf("AddPeriod(1 week): got %d, want %d", got, want)
	}
}

func Test_AddPeriod_Empty(t *testing.T) {
	gd := newGD(500, 240, 0)
	got := gd.AddPeriod("")
	if got != 500 {
		t.Errorf("AddPeriod(''): got %d, want 500", got)
	}
}

func Test_AddPeriod_Noon(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// Start at round 130 (10 rounds past noon). Last noon = 120.
	// "1 noon" = last noon + 1 day = 120 + 240 = 360.
	gd := newGD(130, 240, 0)
	got := gd.AddPeriod("1 noon")
	want := uint64(120 + 240)
	if got != want {
		t.Errorf("AddPeriod(1 noon) from round 130: got %d, want %d", got, want)
	}
}

// --- LastPeriod (month/year) and StartOf ---

func Test_LastPeriod_MatchesPackageWrapper(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// The package-level wrapper must agree with the method for the default calendar.
	for _, name := range []string{"hour", "day", "week", "noon", "sunrise", "sunset"} {
		g := GameDate{Calendar: `default`, RoundNumber: 1700}
		if g.LastPeriod(name) != GetLastPeriod(name, 1700) {
			t.Errorf("LastPeriod(%q) != GetLastPeriod(%q): %d vs %d",
				name, name, g.LastPeriod(name), GetLastPeriod(name, 1700))
		}
	}
}

func Test_LastPeriod_Year(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// 240 rounds/day, 365 days/year => year start at round 0 for year 1.
	// Pick a round mid-year (day 100, 50 rounds in) and confirm it snaps to round 0.
	round := uint64(99*240 + 50)
	g := GameDate{Calendar: `default`, RoundNumber: round}
	if got := g.LastPeriod("year"); got != 0 {
		t.Errorf("LastPeriod(year) from round %d: got %d, want 0", round, got)
	}

	// A round early in year 2 should snap to the first round of year 2.
	roundY2 := uint64(365*240 + 30)
	g2 := GameDate{Calendar: `default`, RoundNumber: roundY2}
	if got := g2.LastPeriod("year"); got != uint64(365*240) {
		t.Errorf("LastPeriod(year) from round %d: got %d, want %d", roundY2, got, 365*240)
	}
}

func Test_LastPeriod_Month(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	// Snapping to the start of a month must land on a midnight whose month matches
	// the starting round's month, and the round just before it must be a different month.
	round := uint64(40 * 240) // day 41, midnight
	g := GameDate{Calendar: `default`, RoundNumber: round}
	start := g.LastPeriod("month")

	startMonth := GetDate(start).Month
	if GetDate(round).Month != startMonth {
		t.Fatalf("month start month %d does not match origin month %d", startMonth, GetDate(round).Month)
	}
	if start%240 != 0 {
		t.Errorf("month start round %d is not aligned to midnight", start)
	}
	if start > 0 && GetDate(start-240).Month == startMonth {
		t.Errorf("round before month start is still month %d; not the true start", startMonth)
	}
}

func Test_StartOf_NormalizesConnectiveWords(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	g := GameDate{Calendar: `default`, RoundNumber: 255}
	want := g.LastPeriod("hour") // 250

	// All of these phrasings must resolve to the same start-of-hour round.
	for _, phrase := range []string{"hour", "start hour", "start of hour", "start of the hour"} {
		if got := g.StartOf(phrase); got != want {
			t.Errorf("StartOf(%q): got %d, want %d", phrase, got, want)
		}
	}
}

func Test_StartOf_NeverMovesForward(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	g := GameDate{Calendar: `default`, RoundNumber: 1234}
	for _, name := range []string{"hour", "day", "week", "month", "year"} {
		if got := g.StartOf(name); got > g.RoundNumber {
			t.Errorf("StartOf(%q) = %d moved forward past %d", name, got, g.RoundNumber)
		}
	}
}

func Test_StartOf_UnknownReturnsUnchanged(t *testing.T) {
	seedCalendarConfig(t, defaultTestCalendar(240, 0))

	g := GameDate{Calendar: `default`, RoundNumber: 777}
	if got := g.StartOf(""); got != 777 {
		t.Errorf("StartOf(empty): got %d, want 777", got)
	}
	if got := g.StartOf("start of the"); got != 777 {
		t.Errorf("StartOf(connectives only): got %d, want 777", got)
	}
}

// --- applyCalendarConfigInto clamp tests ---
func Test_ApplyCalendarConfig_ClampsLowRoundsPerDay(t *testing.T) {
	// rounds_per_day values below 24 must be replaced with the failover value
	// so that roundsPerHour >= 1 and GetLastPeriod never divides by zero.
	for _, rpd := range []int{0, 1, 10, 23} {
		applyCalendarConfig(`default`, CalendarConfig{
			RoundsPerDay: rpd,
			NightHours:   8,
			DuskHours:    3,
			DaysPerYear:  365,
			DaysPerWeek:  7,
			Months:       failoverConfig.Months,
			Zodiac:       failoverConfig.Zodiac,
		})
		ac := activeCalendar[`default`]
		if ac.roundsPerDay < 24 {
			t.Errorf("rounds_per_day=%d: derived roundsPerDay=%d, want >= 24", rpd, ac.roundsPerDay)
		}
		// Must not panic.
		GetLastPeriod(`hour`, 1000)
	}
}

func Test_ApplyCalendarConfig_ClampsNightHours(t *testing.T) {
	// NightHours >= 24 must be clamped to 23 to keep nightStart >= 1.
	applyCalendarConfig(`default`, CalendarConfig{
		RoundsPerDay: 240,
		NightHours:   24,
		DaysPerYear:  365,
		DaysPerWeek:  7,
		Months:       failoverConfig.Months,
		Zodiac:       failoverConfig.Zodiac,
	})
	ac := activeCalendar[`default`]
	if ac.nightHours >= 24 {
		t.Errorf("nightHours=%d, want < 24", ac.nightHours)
	}
}

// --- Benchmarks (preserved from original) ---

func Benchmark_GetDate_Uncached(b *testing.B) {
	util.IncrementRoundCount()
	for n := 0; n < b.N; n++ {
		getDate(uint64(n))
	}
}

func Benchmark_GetDate_Cached(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GetDate()
	}
}
