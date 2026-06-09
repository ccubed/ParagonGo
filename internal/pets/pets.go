package pets

import (
	"fmt"
	"os"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

type Pet struct {
	Name             string       `yaml:"name,omitempty"`             // Name of the pet (player provided hopefully)
	NameStyle        string       `yaml:"namestyle,omitempty"`        // Optional color pattern to apply
	Type             string       `yaml:"type"`                       // type of pet
	RoundActChance   int          `yaml:"roundactchance,omitempty"`   // 0-100 chance per round to fire PetAct script
	Food             Food         `yaml:"food,omitempty"`             // how much food the pet has
	Level            int          `yaml:"level,omitempty"`            // Pet level (1-10)
	LastMealRound    uint8        `yaml:"lastmealround,omitempty"`    // When the pet was last fed
	LastLevelCheck   string       `yaml:"lastlevelcheck,omitempty"`   // "{year}.{day}" of last daily tick
	Abilities        []PetAbility `yaml:"abilities,omitempty"`        // Refreshed from definition file on Validate()
	Items            []items.Item `yaml:"items,omitempty"`            // Items held by this pet
	MissingCountdown int          `yaml:"missingcountdown,omitempty"` // When non-zero, pet is absent

	cachedAbility *PetAbility `yaml:"-"` // cached current ability
	cachedLevel   int         `yaml:"-"` // level when cache was set
}

var (
	petTypes = map[string]*Pet{}
)

func (p *Pet) GetCurrentAbility() *PetAbility {
	if p.cachedAbility != nil && p.cachedLevel == p.Level {
		return p.cachedAbility
	}

	var best *PetAbility
	bestLevel := 0
	for i := range p.Abilities {
		if p.Abilities[i].LevelGranted <= p.Level && p.Abilities[i].LevelGranted >= bestLevel {
			best = &p.Abilities[i]
			bestLevel = p.Abilities[i].LevelGranted
		}
	}

	p.cachedAbility = best
	p.cachedLevel = p.Level
	return best
}

func (p *Pet) clearAbilityCache() {
	p.cachedAbility = nil
	p.cachedLevel = 0
}

func (p *Pet) LevelChange(delta int) (oldLevel int, newLevel int, changed bool) {
	oldLevel = p.Level
	p.Level += delta
	if p.Level < 1 {
		p.Level = 1
	}
	if p.Level > 10 {
		p.Level = 10
	}
	if p.Level != oldLevel {
		p.clearAbilityCache()
		return oldLevel, p.Level, true
	}
	return oldLevel, p.Level, false
}

func (p *Pet) StatMod(statName string) int {
	return p.GetEffectiveStatMods().Get(statName)
}

func (p *Pet) Exists() bool {
	return p.Type != ``
}

// IsMissing returns true when the pet is temporarily absent.
func (p *Pet) IsMissing() bool {
	return p.MissingCountdown > 0
}

// GoMissing sets the missing countdown to the given number of rounds.
// A value of zero clears the missing state immediately.
func (p *Pet) GoMissing(rounds int) {
	if rounds < 0 {
		rounds = 0
	}
	p.MissingCountdown = rounds
}

// DecrementMissing decrements the missing countdown by one.
// Returns true if the countdown just reached zero (pet is returning this round).
func (p *Pet) DecrementMissing() bool {
	if p.MissingCountdown <= 0 {
		return false
	}
	p.MissingCountdown--
	return p.MissingCountdown == 0
}

func (p *Pet) DisplayName() string {

	name := p.Name
	if name == `` {
		name = p.Type
	}

	result := ``
	if len(p.NameStyle) > 0 {
		patternName := p.NameStyle
		if patternName[0:1] == `:` {
			patternName = patternName[1:]
		}
		result = colorpatterns.ApplyColorPattern(name, patternName)
	} else {
		result = fmt.Sprintf(`<ansi fg="petname">%s</ansi>`, name)
	}

	if p.Food == 0 {
		result += ` <ansi fg="alert-2">(Starving)</ansi>`
	} else if p.Food == 1 {
		result += ` <ansi fg="alert-1">(Hungry)</ansi>`
	}

	return result
}

func (p *Pet) StoreItem(i items.Item) bool {

	if p.GetEffectiveCapacity() < 1 {
		return false
	}

	if i.ItemId < 1 {
		return false
	}
	i.Validate()
	p.Items = append(p.Items, i)
	return true
}

func (p *Pet) RemoveItem(i items.Item) bool {

	for j := len(p.Items) - 1; j >= 0; j-- {
		if p.Items[j].Equals(i) {
			p.Items = append(p.Items[:j], p.Items[j+1:]...)
			return true
		}
	}
	return false
}

func (p *Pet) GetBuffs() []int {
	return p.GetEffectiveBuffs()
}

func (p *Pet) FindItem(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, p.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (p *Pet) GetDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {
	_, d := p.GetEffectiveDamage()
	return d.Attacks, d.DiceCount, d.SideCount, d.BonusDamage, d.CritBuffIds
}

// GetCombatMessages returns the effective combat messages for the current
// ability level. Any empty slot in the ability's AttackMessages is filled in
// from the built-in defaults. targetType is the ANSI fg colour prefix (e.g.
// "mob" or "user") used when building the default fallback strings.
func (p *Pet) GetCombatMessages(targetType string) CombatMessages {
	defaults := DefaultCombatMessages(targetType)

	var custom CombatMessages
	if ab := p.GetCurrentAbility(); ab != nil {
		custom = ab.AttackMessages
	}

	result := CombatMessages{
		ToOwner:  custom.ToOwner,
		ToTarget: custom.ToTarget,
		ToRoom:   custom.ToRoom,
		Miss:     custom.Miss,
	}
	if result.ToOwner == `` {
		result.ToOwner = defaults.ToOwner
	}
	if result.ToTarget == `` {
		result.ToTarget = defaults.ToTarget
	}
	if result.ToRoom == `` {
		result.ToRoom = defaults.ToRoom
	}
	if result.Miss == `` {
		result.Miss = defaults.Miss
	}
	return result
}

func (p *Pet) GetEffectiveStatMods() statmods.StatMods {
	result := make(statmods.StatMods)
	if ab := p.GetCurrentAbility(); ab != nil {
		for k, v := range ab.StatMods {
			result.Add(k, v)
		}
	}
	return result
}

func (p *Pet) GetEffectiveCapacity() int {
	if ab := p.GetCurrentAbility(); ab != nil {
		return ab.Capacity
	}
	return 0
}

func (p *Pet) GetEffectiveDamage() (int, items.Damage) {
	if ab := p.GetCurrentAbility(); ab != nil && ab.Damage.DiceRoll != `` {
		return ab.CombatChance, ab.Damage
	}
	return 0, items.Damage{}
}

func (p *Pet) GetEffectiveBuffs() []int {
	if ab := p.GetCurrentAbility(); ab != nil && len(ab.BuffIds) > 0 {
		return append([]int{}, ab.BuffIds...)
	}
	return []int{}
}

func (p *Pet) DailyLevelCheck() int {
	if p.Food == 3 {
		_, _, changed := p.LevelChange(1)
		if changed {
			return 1
		}
	}
	if p.Food == 0 {
		_, _, changed := p.LevelChange(-1)
		if changed {
			return -1
		}
	}
	return 0
}

func (p *Pet) GetCurrentDayKey() string {
	gd := gametime.GetDate()
	return fmt.Sprintf(`%d.%d`, gd.Year, gd.Day)
}

// CheckDailyTick returns (levelChange, needsValidate) if a new day has passed.
// Returns (0, false) if no tick is needed.
func (p *Pet) CheckDailyTick() (int, bool) {
	dayKey := p.GetCurrentDayKey()
	if p.LastLevelCheck == dayKey {
		return 0, false
	}

	p.LastLevelCheck = dayKey

	levelChange := p.DailyLevelCheck()
	p.Food.Remove()

	return levelChange, true
}

type AbilityDisplay struct {
	LevelGranted int
	Active       bool
	CombatChance int
	DiceRoll     string
	DiceCount    int
	SideCount    int
	StatMods     map[string]int
	BuffNames    []string
	Capacity     int
}

func (p *Pet) GetAbilityDisplays() []AbilityDisplay {
	result := []AbilityDisplay{}
	for _, a := range p.Abilities {
		d := AbilityDisplay{
			LevelGranted: a.LevelGranted,
			Active:       a.LevelGranted <= p.Level,
			CombatChance: a.CombatChance,
			DiceRoll:     a.Damage.DiceRoll,
			DiceCount:    a.Damage.DiceCount,
			SideCount:    a.Damage.SideCount,
			StatMods:     map[string]int(a.StatMods),
			Capacity:     a.Capacity,
		}
		for _, bId := range a.BuffIds {
			name := fmt.Sprintf(`#%d`, bId)
			if spec := buffs.GetBuffSpec(bId); spec != nil {
				name = spec.Name
			}
			d.BuffNames = append(d.BuffNames, name)
		}
		result = append(result, d)
	}
	return result
}

func (p *Pet) GetCurrentAbilityDisplay() *AbilityDisplay {
	ab := p.GetCurrentAbility()
	if ab == nil {
		return nil
	}
	d := &AbilityDisplay{
		LevelGranted: ab.LevelGranted,
		Active:       true,
		CombatChance: ab.CombatChance,
		DiceRoll:     ab.Damage.DiceRoll,
		DiceCount:    ab.Damage.DiceCount,
		SideCount:    ab.Damage.SideCount,
		StatMods:     map[string]int(ab.StatMods),
		Capacity:     ab.Capacity,
	}
	for _, bId := range ab.BuffIds {
		name := fmt.Sprintf(`#%d`, bId)
		if spec := buffs.GetBuffSpec(bId); spec != nil {
			name = spec.Name
		}
		d.BuffNames = append(d.BuffNames, name)
	}
	return d
}

func GetPetCopy(petId string) Pet {
	if petInfo, ok := petTypes[petId]; ok {
		return *petInfo
	}
	return Pet{}
}

func (p *Pet) Filename() string {
	filename := util.ConvertForFilename(p.Type)
	return fmt.Sprintf("%s.yaml", filename)
}

func (p *Pet) Filepath() string {
	return p.Filename()
}

func (p *Pet) Save() error {
	bytes, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `pets`, `/`, p.Filename())

	err = util.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pet) Id() string {
	return p.Type
}

func (p *Pet) GetScriptPath() string {
	// Prefers .js, falls back to .lua
	return util.ResolveScriptPath(util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `pets`, `/`, p.Filepath()))
}

func (p *Pet) HasScript() bool {
	if script := getPluginScript(p.Type); script != `` {
		return true
	}
	scriptPath := p.GetScriptPath()
	_, err := os.Stat(scriptPath)
	return err == nil
}

func (p *Pet) GetScript() string {
	// Check plugin-registered scripts first.
	if script := getPluginScript(p.Type); script != `` {
		return script
	}
	scriptPath := p.GetScriptPath()
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := util.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}
	return ``
}

func (p *Pet) Validate() error {

	if p.Items == nil {
		p.Items = []items.Item{}
	}

	if p.Food > 3 {
		p.Food = 3
	}
	if p.Food < 0 {
		p.Food = 0
	}

	if p.Exists() && p.Level < 1 {
		p.Level = 1
	}
	if p.Level > 10 {
		p.Level = 10
	}

	if p.Exists() {
		if def, ok := petTypes[p.Type]; ok {
			p.Abilities = make([]PetAbility, len(def.Abilities))
			copy(p.Abilities, def.Abilities)
			p.NameStyle = def.NameStyle
			p.RoundActChance = def.RoundActChance
			p.clearAbilityCache()
		}
	}

	for i := range p.Abilities {
		p.Abilities[i].Damage.InitDiceRoll(p.Abilities[i].Damage.DiceRoll)
		p.Abilities[i].Damage.FormatDiceRoll()
		if p.Abilities[i].BuffIds == nil {
			p.Abilities[i].BuffIds = []int{}
		}

	}

	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpPetTypes, err := fileloader.LoadAllFlatFiles[string, *Pet](configs.GetFilePathsConfig().DataFiles.String() + `/pets`)
	if err != nil {
		panic(err)
	}

	petTypes = tmpPetTypes

	// Merge pets from plugin file systems.
	loadPluginPets(petTypes)

	mudlog.Info("pets.LoadDataFiles()", "loadedCount", len(petTypes), "Time Taken", time.Since(start))
}
