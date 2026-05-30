package leaderboards

import (
	"embed"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (

	//////////////////////////////////////////////////////////////////////
	// NOTE: The below //go:embed directive is important!
	// It embeds the relative path into the var below it.
	//////////////////////////////////////////////////////////////////////

	//go:embed files/*
	files embed.FS
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {
	t := LeaderboardModule{
		plug: plugins.New(`leaderboards`, `1.0`),
	}

	if err := t.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	t.plug.Web.AdminPage("Config", "leaderboards-config", "html/admin/leaderboards-config.html", true, "Modules", "Leaderboards", "Configure leaderboard categories and scoring rules.", "Leaderboards module tracking and displaying top player rankings across categories.", nil)
	t.plug.Web.AdminPage("About", "leaderboards-about", "html/admin/leaderboards-about.html", true, "Modules", "Leaderboards", "Information and version details for the Leaderboards module.", "", nil)

	t.plug.AddUserCommand(`leaderboard`, t.leaderboardCommand, true, false)

	t.plug.Callbacks.SetOnLoad(t.loadLBs)
	t.plug.Callbacks.SetOnSave(t.saveLBs)

	t.plug.Web.WebPage(`Leaderboards`, `/leaderboards`, `leaderboards.html`, true, t.webLeaderboardData)

	events.RegisterListener(events.NewRound{}, t.newRoundHandler)
	events.RegisterListener(events.PlayerSpawn{}, t.playerSpawnHandler)
	events.RegisterListener(events.PlayerDespawn{}, t.playerDespawnHandler)
	events.RegisterListener(events.LevelUp{}, t.userChangedHandler)
	events.RegisterListener(events.GainExperience{}, t.userChangedHandler)
	events.RegisterListener(events.MobDeath{}, t.mobDeathHandler)
	events.RegisterListener(events.EquipmentChange{}, t.equipmentChangeHandler)
	events.RegisterListener(events.CharacterChanged{}, t.characterChangedHandler)
	events.RegisterListener(events.PlayerDeath{}, t.userChangedHandler)
}

//////////////////////////////////////////////////////////////////////
// NOTE: What follows is all custom code for this module.
//////////////////////////////////////////////////////////////////////

// cachedUserData holds the leaderboard-relevant fields for a single character.
// Alts are nested so one map entry covers an entire account. The struct is
// persisted to a dedicated plugin data file so the full user-directory walk
// only happens once (on first load after the module is installed).
type cachedUserData struct {
	UserId      int              `yaml:"UserId"`
	CharName    string           `yaml:"CharName"`
	CharClass   string           `yaml:"CharClass"`
	Level       int              `yaml:"Level"`
	Gold        int              `yaml:"Gold"`
	Bank        int              `yaml:"Bank"`
	Experience  int              `yaml:"Experience"`
	TotalKills  int              `yaml:"TotalKills"`
	Exploration int              `yaml:"Exploration"`
	Alts        []cachedUserData `yaml:"Alts,omitempty"`
}

// LeaderboardModule holds all state for the leaderboards module.
type LeaderboardModule struct {
	plug *plugins.Plugin

	lastCalculated time.Time

	GoldLBSize        int
	ExperienceLBSize  int
	KillsLBSize       int
	ExplorationLBSize int

	LB_Gold        leaderboardData `yaml:"LB_Gold,omitempty"`
	LB_Experience  leaderboardData `yaml:"LB_Experience,omitempty"`
	LB_Kills       leaderboardData `yaml:"LB_Kills,omitempty"`
	LB_Exploration leaderboardData `yaml:"LB_Exploration,omitempty"`

	// userCache maps userId to the pre-computed leaderboard data for that
	// account (primary character + alts). It is persisted separately under
	// the "user-cache" plugin data identifier so the expensive full user-file
	// disk walk only happens on the very first load.
	userCache map[int]cachedUserData
}

func (l *LeaderboardModule) webLeaderboardData(r *http.Request) map[string]any {
	return map[string]any{
		`leaderboards`: l.getCurrentLeaderboards(),
	}
}

func (l *LeaderboardModule) loadLBs() {
	l.plug.ReadIntoStruct(`latest-leaderboards`, &l)

	l.LB_Gold = leaderboardData{Name: `Gold`, ValueColor: `experience`}
	l.LB_Experience = leaderboardData{Name: `Experience`, ValueColor: `gold`}
	l.LB_Kills = leaderboardData{Name: `Kills`, ValueColor: `red-bold`}
	l.LB_Exploration = leaderboardData{Name: `Exploration`, ValueColor: `cyan-bold`}

	l.userCache = make(map[int]cachedUserData)
	l.plug.ReadIntoStruct(`user-cache`, &l.userCache)

	// If the cache is empty this is the first run; do a one-time cold
	// population from disk so subsequent updates never need to walk user files.
	if len(l.userCache) == 0 {
		l.coldPopulateCache()
	}
}

func (l *LeaderboardModule) saveLBs() {
	l.plug.WriteStruct(`latest-leaderboards`, l)
	l.plug.WriteStruct(`user-cache`, l.userCache)
}

// coldPopulateCache performs a one-time full scan of all users (online and
// offline) to seed the userCache. After this, the cache is kept current via
// event listeners and no further disk walks are needed.
func (l *LeaderboardModule) coldPopulateCache() {
	start := time.Now()

	loadAlts := resolveLoadAlts()

	for _, u := range users.GetAllActiveUsers() {
		l.userCache[u.UserId] = buildCacheEntry(u.UserId, u.Character, loadAlts)
	}

	users.SearchOfflineUsers(func(u *users.UserRecord) bool {
		l.userCache[u.UserId] = buildCacheEntry(u.UserId, u.Character, loadAlts)
		return true
	})

	mudlog.Info("leaderboard.coldPopulateCache()", "users-cached", len(l.userCache), "time-taken", time.Since(start))
}

// resolveLoadAlts looks up the LoadAlts exported function once and returns a
// typed wrapper, or nil if the alt-characters module is not loaded.
func resolveLoadAlts() func(int) []characters.Character {
	fn, ok := usercommands.GetExportedFunction(`LoadAlts`)
	if !ok {
		return nil
	}
	loadAlts, ok := fn.(func(int) []characters.Character)
	if !ok {
		return nil
	}
	return loadAlts
}

// buildCacheEntry constructs a cachedUserData for a single account, including
// all alt characters if the alt-characters module is loaded.
func buildCacheEntry(userId int, char *characters.Character, loadAlts func(int) []characters.Character) cachedUserData {
	entry := cachedUserData{
		UserId:      userId,
		CharName:    char.Name,
		CharClass:   skills.GetProfession(char.GetAllSkillRanks()),
		Level:       char.Level,
		Gold:        char.Gold,
		Bank:        char.Bank,
		Experience:  char.Experience,
		TotalKills:  char.KD.TotalKills,
		Exploration: explorationScore(char),
	}

	if loadAlts != nil {
		for _, alt := range loadAlts(userId) {
			alt := alt
			entry.Alts = append(entry.Alts, cachedUserData{
				UserId:      userId,
				CharName:    alt.Name,
				CharClass:   skills.GetProfession(alt.GetAllSkillRanks()),
				Level:       alt.Level,
				Gold:        alt.Gold,
				Bank:        alt.Bank,
				Experience:  alt.Experience,
				TotalKills:  alt.KD.TotalKills,
				Exploration: explorationScore(&alt),
			})
		}
	}

	return entry
}

// refreshCacheEntry rebuilds the cache entry for userId from the live
// in-memory UserRecord. Called by event handlers; no disk I/O.
func (l *LeaderboardModule) refreshCacheEntry(userId int) {
	u := users.GetByUserId(userId)
	if u == nil {
		return
	}
	l.userCache[userId] = buildCacheEntry(u.UserId, u.Character, resolveLoadAlts())
}

func explorationScore(char *characters.Character) int {
	total := 0
	for _, bs := range char.ZonesVisited {
		total += bs.Count()
	}
	return total
}

func (l *LeaderboardModule) leaderboardCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	for _, lb := range l.getCurrentLeaderboards() {

		title := fmt.Sprintf(`%s Leaderboard`, lb.Name)

		headers := []string{`Rank`, `Character`, `Profession`, `Level`, lb.Name}

		rows := [][]string{}

		valueFormatting := `%s`
		if lb.ValueColor != `` {
			valueFormatting = `<ansi fg="` + lb.ValueColor + `">%s</ansi>`
		}

		formatting := []string{
			`<ansi fg="red">%s</ansi>`,
			`<ansi fg="username">%s</ansi>`,
			`<ansi fg="white-bold">%s</ansi>`,
			`<ansi fg="157">%s</ansi>`,
			valueFormatting,
		}

		for i, entry := range lb.Top {

			if entry.UserId == 0 {
				continue
			}

			newRow := []string{`#` + strconv.Itoa(i+1), entry.CharacterName, entry.CharacterClass, strconv.Itoa(entry.Level), util.FormatNumber(entry.ScoreValue)}

			rows = append(rows, newRow)
		}

		searchResultsTable := templates.GetTable(title, headers, rows, formatting)
		tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId)
		user.SendText("\n")
		user.SendText(tplTxt)
	}
	return true, nil
}

func (l *LeaderboardModule) Reset() {
	l.LB_Gold.Reset(l.GoldLBSize)
	l.LB_Experience.Reset(l.ExperienceLBSize)
	l.LB_Kills.Reset(l.KillsLBSize)
	l.LB_Exploration.Reset(l.ExplorationLBSize)
}

func (l *LeaderboardModule) RefreshConfig() {

	l.GoldLBSize = 10
	if size, ok := l.plug.Config.Get(`GoldLBSize`).(int); ok {
		l.GoldLBSize = size
	}

	l.ExperienceLBSize = 10
	if size, ok := l.plug.Config.Get(`ExperienceLBSize`).(int); ok {
		l.ExperienceLBSize = size
	}

	l.KillsLBSize = 10
	if size, ok := l.plug.Config.Get(`KillsLBSize`).(int); ok {
		l.KillsLBSize = size
	}

	l.ExplorationLBSize = 10
	if size, ok := l.plug.Config.Get(`ExplorationLBSize`).(int); ok {
		l.ExplorationLBSize = size
	}
}

func (l *LeaderboardModule) Update() {
	start := time.Now()

	l.Reset()

	for _, entry := range l.userCache {
		l.considerEntry(entry)
		for _, alt := range entry.Alts {
			l.considerEntry(alt)
		}
	}

	mudlog.Info("leaderboard.Update()", "cached-users", len(l.userCache), "Time Taken", time.Since(start))

	l.lastCalculated = time.Now()
}

func (l *LeaderboardModule) considerEntry(entry cachedUserData) {
	if l.GoldLBSize > 0 {
		l.LB_Gold.Consider(entry.UserId, entry.CharName, entry.CharClass, entry.Level, entry.Gold+entry.Bank)
	}
	if l.ExperienceLBSize > 0 {
		l.LB_Experience.Consider(entry.UserId, entry.CharName, entry.CharClass, entry.Level, entry.Experience)
	}
	if l.KillsLBSize > 0 {
		l.LB_Kills.Consider(entry.UserId, entry.CharName, entry.CharClass, entry.Level, entry.TotalKills)
	}
	if l.ExplorationLBSize > 0 {
		l.LB_Exploration.Consider(entry.UserId, entry.CharName, entry.CharClass, entry.Level, entry.Exploration)
	}
}

func (l *LeaderboardModule) newRoundHandler(e events.Event) events.ListenerReturn {
	if time.Since(l.lastCalculated).Minutes() >= 5 {
		l.Update()
	}
	return events.Continue
}

func (l *LeaderboardModule) playerSpawnHandler(e events.Event) events.ListenerReturn {
	evt := e.(events.PlayerSpawn)
	l.refreshCacheEntry(evt.UserId)
	return events.Continue
}

func (l *LeaderboardModule) playerDespawnHandler(e events.Event) events.ListenerReturn {
	evt := e.(events.PlayerDespawn)
	l.refreshCacheEntry(evt.UserId)
	return events.Continue
}

func (l *LeaderboardModule) userChangedHandler(e events.Event) events.ListenerReturn {
	var userId int
	switch evt := e.(type) {
	case events.LevelUp:
		userId = evt.UserId
	case events.GainExperience:
		userId = evt.UserId
	case events.PlayerDeath:
		userId = evt.UserId
	default:
		return events.Continue
	}
	l.refreshCacheEntry(userId)
	return events.Continue
}

func (l *LeaderboardModule) mobDeathHandler(e events.Event) events.ListenerReturn {
	evt := e.(events.MobDeath)
	for _, userId := range evt.KilledByUsers {
		l.refreshCacheEntry(userId)
	}
	return events.Continue
}

func (l *LeaderboardModule) equipmentChangeHandler(e events.Event) events.ListenerReturn {
	evt := e.(events.EquipmentChange)
	if evt.UserId != 0 {
		l.refreshCacheEntry(evt.UserId)
	}
	return events.Continue
}

func (l *LeaderboardModule) characterChangedHandler(e events.Event) events.ListenerReturn {
	evt := e.(events.CharacterChanged)
	l.refreshCacheEntry(evt.UserId)
	return events.Continue
}

func (l *LeaderboardModule) getCurrentLeaderboards() []leaderboardData {

	l.RefreshConfig()

	if l.lastCalculated.IsZero() {
		l.Update()
	}

	ret := []leaderboardData{}

	if l.GoldLBSize > 0 {
		ret = append(ret, l.LB_Gold)
	}

	if l.ExperienceLBSize > 0 {
		ret = append(ret, l.LB_Experience)
	}

	if l.KillsLBSize > 0 {
		ret = append(ret, l.LB_Kills)
	}

	if l.ExplorationLBSize > 0 {
		ret = append(ret, l.LB_Exploration)
	}

	return ret
}

type leaderboardEntry struct {
	UserId         int    `yaml:"UserId,omitempty"`
	CharacterName  string `yaml:"CharacterName,omitempty"`
	CharacterClass string `yaml:"CharacterClass,omitempty"`
	Level          int    `yaml:"Level,omitempty"`
	ScoreValue     int    `yaml:"ScoreValue,omitempty"`
}

type leaderboardData struct {
	Name        string
	ValueColor  string
	Top         []leaderboardEntry `yaml:"Top,omitempty"`
	MaxSize     int
	LowestValue int
}

func (l *leaderboardData) Reset(size int) {
	l.MaxSize = size
	if size > 0 {
		l.Top = make([]leaderboardEntry, l.MaxSize)
	} else {
		l.Top = nil
	}
	l.LowestValue = 0
}

func (l *leaderboardData) Consider(userId int, charName, charClass string, level, val int) {
	if val == 0 {
		return
	}

	if val < l.LowestValue && l.Top[l.MaxSize-1].UserId != 0 {
		return
	}

	addPosition := -1
	for i := 0; i < l.MaxSize; i++ {

		if l.Top[i].UserId == 0 {
			addPosition = i
			break
		}

		if val > l.Top[i].ScoreValue {
			addPosition = i
			break
		}
	}

	if addPosition > -1 {

		for i := l.MaxSize - 2; i >= addPosition; i-- {
			l.Top[i+1] = l.Top[i]
		}

		l.Top[addPosition] = leaderboardEntry{
			UserId:         userId,
			CharacterName:  charName,
			CharacterClass: charClass,
			Level:          level,
			ScoreValue:     val,
		}

		if l.LowestValue == 0 || val < l.LowestValue {
			l.LowestValue = val
		}
	}
}
