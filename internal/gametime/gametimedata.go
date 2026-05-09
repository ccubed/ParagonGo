package gametime

import (
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert/yaml"
)

var (
	gameTimeData = CalendarOptions{
		Calendars: make(map[string]CalendarConfig),
	}

	// failoverConfig is used when no valid data could be loaded
	failoverConfig = CalendarConfig{
		RoundsPerDay: 900,
		NightHours:   8,
		DuskHours:    3,
		DaysPerYear:  365,
		DaysPerWeek:  7,
		SunCount:     1,
		MoonCount:    1,
		Months:       []string{`January`, `February`, `March`, `April`, `May`, `June`, `July`, `August`, `September`, `November`, `December`},
		Zodiac:       []string{`Rat`, `Ox`, `Tiger`, `Rabbit`, `Dragon`, `Snake`, `Horse`, `Goat`, `Monkey`, `Rooster`, `Dog`, `Pig`},
	}

	// activeCalendar holds pre-computed derived constants for every named calendar.
	// Keyed by calendar name (e.g. "default"). All hot-path functions index by name —
	// no locks, no config lookups per call.
	// Seeded with failoverConfig so any package that calls into gametime before
	// LoadGameTimeConfigs runs (e.g. via a package-level var initializer) never
	// receives a zero-value struct with RoundsPerDay == 0.
	activeCalendar = func() map[string]calendarDerived {
		m := map[string]calendarDerived{}
		applyCalendarConfigInto(m, `default`, failoverConfig)
		return m
	}()
)

// calendarDerived holds pre-computed constants derived from a CalendarConfig.
// It is populated once at startup (and in tests) so that getDate, GetLastPeriod,
// and ReCalculate never need to recompute or lock-acquire anything.
type calendarDerived struct {
	roundsPerDay    uint64
	roundsPerWeek   uint64
	roundsPerHour   float64
	roundsPerMinute float64
	noonRound       uint64 // roundsPerDay / 2
	nightHours      int
	nightStart      int // 24 - halfNight  (hour when night begins)
	nightEnd        int // nightHours - halfNight  (hour when day begins, i.e. DayStart)
	duskHours       int // how many hours before nightStart dusk colour applies
	daysPerYear     int
	daysPerWeek     int
	numMonths       int
	hoursPerMonth   float64 // (daysPerYear * 24) / numMonths
	sunCount        int     // 1-2
	moonCount       int     // 0-3
	monthNames      []string
	zodiacList      []string // shuffled
}

// CalendarConfig holds the configuration for a single named calendar system.
type CalendarConfig struct {
	RoundsPerDay int      `yaml:"rounds_per_day" json:"rounds_per_day"`
	NightHours   int      `yaml:"night_hours"    json:"night_hours"`
	DuskHours    int      `yaml:"dusk_hours"     json:"dusk_hours"`
	DaysPerYear  int      `yaml:"days_per_year"  json:"days_per_year"`
	DaysPerWeek  int      `yaml:"days_per_week"  json:"days_per_week"`
	SunCount     int      `yaml:"sun_count"      json:"sun_count"`  // 1-2
	MoonCount    int      `yaml:"moon_count"     json:"moon_count"` // 0-3
	Months       []string `yaml:"months"         json:"months"`
	Zodiac       []string `yaml:"zodiac"         json:"zodiac"`
}

// CalendarOptions is the top-level structure for gametime.yaml.
type CalendarOptions struct {
	Calendars map[string]CalendarConfig `yaml:"calendars"`
}

func (o CalendarOptions) GetCalendarConfig(name string) CalendarConfig {
	if c, ok := o.Calendars[name]; ok {
		return c
	}
	return o.Calendars[`default`]
}

func LoadGameTimeConfigs() {

	start := time.Now()

	path := util.FilePath(string(configs.GetFilePathsConfig().DataFiles) + `/gametime.yaml`)

	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+path))
	}

	err = yaml.Unmarshal(bytes, &gameTimeData)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+path))
	}

	if gameTimeData.Calendars == nil {
		gameTimeData.Calendars = make(map[string]CalendarConfig)
	}

	if _, ok := gameTimeData.Calendars[`default`]; !ok {
		gameTimeData.Calendars[`default`] = failoverConfig
	}

	for name, cfg := range gameTimeData.Calendars {
		applyCalendarConfig(name, cfg)
	}

	mudlog.Info("...LoadGameTimeConfigs()", "loadedCount", len(gameTimeData.Calendars), "Time Taken", time.Since(start))
}

// applyCalendarConfig pre-computes all derived constants from c and stores them
// in activeCalendar[name]. It also resets the round-date cache when the "default"
// calendar changes, since that is what drives getDate. Safe to call from tests.
func applyCalendarConfig(name string, c CalendarConfig) {
	applyCalendarConfigInto(activeCalendar, name, c)
	if name == `default` {
		clear(roundDateCache)
		roundDateCacheSeq = roundDateCacheSeq[:0]
	}
}

// applyCalendarConfigInto is the pure computation used by both applyCalendarConfig
// and the activeCalendar initializer. It does not touch the cache.
func applyCalendarConfigInto(m map[string]calendarDerived, name string, c CalendarConfig) {

	if c.RoundsPerDay < 24 {
		c.RoundsPerDay = failoverConfig.RoundsPerDay
	}
	if c.NightHours < 0 {
		c.NightHours = 0
	} else if c.NightHours >= 24 {
		c.NightHours = 23
	}
	if c.DaysPerYear < 1 {
		c.DaysPerYear = failoverConfig.DaysPerYear
	}
	if c.DaysPerWeek < 1 {
		c.DaysPerWeek = failoverConfig.DaysPerWeek
	}
	if c.SunCount < 1 {
		c.SunCount = 1
	} else if c.SunCount > 2 {
		c.SunCount = 2
	}
	if c.MoonCount < 0 {
		c.MoonCount = 0
	} else if c.MoonCount > 3 {
		c.MoonCount = 3
	}
	if len(c.Months) == 0 {
		c.Months = failoverConfig.Months
	}
	if len(c.Zodiac) == 0 {
		c.Zodiac = failoverConfig.Zodiac
	}

	rpd := uint64(c.RoundsPerDay)
	rph := float64(rpd) / 24
	halfNight := int(math.Floor(float64(c.NightHours) / 2))
	numMonths := len(c.Months)

	hoursPerMonth := float64(0)
	if numMonths > 0 {
		hoursPerMonth = float64(c.DaysPerYear) * 24 / float64(numMonths)
	}

	// Copy and shuffle zodiac list using the world seed so the order is
	// deterministic per server instance (same as the previous randomizeZodiac logic).
	zodiac := make([]string, len(c.Zodiac))
	copy(zodiac, c.Zodiac)
	r := rand.New(rand.NewSource(configs.GetConfig().SeedInt()))
	r.Shuffle(len(zodiac), func(i, j int) { zodiac[i], zodiac[j] = zodiac[j], zodiac[i] })

	// Copy month names
	months := make([]string, len(c.Months))
	copy(months, c.Months)

	m[name] = calendarDerived{
		roundsPerDay:    rpd,
		roundsPerWeek:   rpd * uint64(c.DaysPerWeek),
		roundsPerHour:   rph,
		roundsPerMinute: rph / 60,
		noonRound:       rpd / 2,
		nightHours:      c.NightHours,
		nightStart:      24 - halfNight,
		nightEnd:        c.NightHours - halfNight,
		duskHours:       c.DuskHours,
		daysPerYear:     c.DaysPerYear,
		daysPerWeek:     c.DaysPerWeek,
		numMonths:       numMonths,
		hoursPerMonth:   hoursPerMonth,
		sunCount:        c.SunCount,
		moonCount:       c.MoonCount,
		monthNames:      months,
		zodiacList:      zodiac,
	}
}
