package gametime

import (
	"fmt"
	"os"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// GetCalendars returns a sorted slice of calendar names from the loaded data.
func GetCalendars() []string {
	names := make([]string, 0, len(gameTimeData.Calendars))
	for name := range gameTimeData.Calendars {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetCalendar returns the CalendarConfig for the named calendar, plus a bool
// indicating whether it was found.
func GetCalendar(name string) (CalendarConfig, bool) {
	c, ok := gameTimeData.Calendars[name]
	return c, ok
}

// SaveCalendar upserts a named calendar into the in-memory store, applies the
// derived constants, and persists the full gametime.yaml to disk.
func SaveCalendar(name string, cfg CalendarConfig) error {
	if name == "" {
		return fmt.Errorf("calendar name is required")
	}
	if cfg.RoundsPerDay < 24 {
		return fmt.Errorf("rounds_per_day must be at least 24")
	}
	if cfg.DaysPerYear < 1 {
		return fmt.Errorf("days_per_year must be at least 1")
	}
	if cfg.DaysPerWeek < 1 {
		return fmt.Errorf("days_per_week must be at least 1")
	}
	if len(cfg.Months) == 0 {
		return fmt.Errorf("months list cannot be empty")
	}
	if len(cfg.Zodiac) == 0 {
		return fmt.Errorf("zodiac list cannot be empty")
	}

	gameTimeData.Calendars[name] = cfg
	applyCalendarConfig(name, cfg)

	if err := saveGameTimeFile(); err != nil {
		return err
	}

	mudlog.Info("gametime.SaveCalendar", "calendar", name)
	return nil
}

// DeleteCalendar removes a named calendar from the in-memory store and
// persists the change to disk. The "default" calendar cannot be deleted.
func DeleteCalendar(name string) error {
	if name == "default" {
		return fmt.Errorf("the default calendar cannot be deleted")
	}
	if _, ok := gameTimeData.Calendars[name]; !ok {
		return fmt.Errorf("calendar %q not found", name)
	}

	delete(gameTimeData.Calendars, name)
	delete(activeCalendar, name)

	if err := saveGameTimeFile(); err != nil {
		return err
	}

	mudlog.Info("gametime.DeleteCalendar", "calendar", name)
	return nil
}

// saveGameTimeFile writes the current gameTimeData to the gametime.yaml path.
func saveGameTimeFile() error {
	path := util.FilePath(string(configs.GetFilePathsConfig().DataFiles) + `/gametime.yaml`)

	data, err := yaml.Marshal(&gameTimeData)
	if err != nil {
		return errors.Wrap(err, "marshal gametime data")
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return errors.Wrap(err, "write gametime.yaml: "+path)
	}

	return nil
}
