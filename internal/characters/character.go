package characters

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/races"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/util"

	//
	"maps"
	"slices"
)

var (
	startingRace   = 0
	startingHealth = 10
	startingMana   = 10
	StartingRoomId = -1
	startingZone   = `Nowhere`
	defaultName    = `nameless`
)

type NameRenderFlag uint8

const (
	RenderHealth NameRenderFlag = iota
	RenderAggro
	RenderShortAdjectives
)

type Character struct {
	Name                string                         // The name of the character
	Description         string                         // A description of the character.
	Adjectives          []string                       `yaml:"adjectives,omitempty"` // Decorative text for the name of the character (e.g. "sleeping", "dead", "wounded")
	RoomId              int                            // The room id the character is in.
	RoomIdOnReset       int                            // The room they are sent to if their RoomId isn't found.
	Zone                string                         // The zone the character is in. The folder the room can be located in too.
	RaceId              int                            // Character race
	FormRaceId          int                            `yaml:"formraceid,omitempty"` // Temporary race override (0 = not transformed)
	Stats               stats.Statistics               // Character stats
	Level               int                            // The level of the character
	Experience          int                            // The experience of the character
	TrainingPoints      int                            // The number of training points the character has
	StatPoints          int                            // The number of skill points the character has
	Health              int                            // The health of the character
	Mana                int                            // The mana of the character
	ActionPoints        int                            // The resevoir of action points the character has to spend on movement etc.
	Alignment           int8                           // The alignment of the character
	Gold                int                            // The gold the character is holding
	Bank                int                            // The gold the character has in the bank
	Shop                Shop                           `yaml:"shop,omitempty"`          // Definition of shop services/items this character stocks (or just has at the moment)
	SpellBook           map[string]int                 `yaml:"spellbook,omitempty"`     // The spells the character has learned
	Charmed             *CharmInfo                     `yaml:"-"`                       // If they are charmed, this is the info
	CharmedMobs         []int                          `yaml:"-"`                       // If they have charmed anyone, this is the list of mob instance ids
	Items               []items.Item                   `yaml:"items,omitempty"`         // The items the character is holding
	Buffs               buffs.Buffs                    `yaml:"buffs,omitempty"`         // The buffs the character has active
	Equipment           Worn                           `yaml:"equipment,omitempty"`     // The equipment the character is wearing
	TNLScale            float32                        `yaml:"-"`                       // The experience scale of the character. Don't write to yaml since is dynamically calculated.
	HealthMax           stats.StatInfo                 `yaml:"-"`                       // The maximum health of the character. Don't write to yaml since is dynamically calculated.
	ManaMax             stats.StatInfo                 `yaml:"-"`                       // The maximum mana of the character. Don't write to yaml since is dynamically calculated.
	ActionPointsMax     stats.StatInfo                 `yaml:"-"`                       // The maximum actions of character. Don't write to yaml since is dynamically calculated.
	Aggro               *Aggro                         `yaml:"-"`                       // Dont' store this. If they leave they break their aggro
	Skills              map[string]int                 `yaml:"skills,omitempty"`        // The skills the character has, and what level they are at
	Cooldowns           Cooldowns                      `yaml:"cooldowns,omitempty"`     // How many rounds until it is cooled down
	Settings            map[string]string              `yaml:"settings,omitempty"`      // custom setting tracking, used for anything.
	QuestProgress       map[int]string                 `yaml:"questprogress,omitempty"` // quest progress tracking
	KeyRing             map[string]string              `yaml:"keyring,omitempty"`       // key is the lock id, value is the sequence
	KD                  KDStats                        `yaml:"kd,omitempty"`            // Kill/Death stats
	MiscData            map[string]any                 `yaml:"miscdata,omitempty"`      // Any random other data that needs to be stored
	ExtraLives          int                            `yaml:"extralives,omitempty"`    // How many lives remain. If enabled, players can perma-die if they die at zero
	MobMastery          MobMasteries                   `yaml:"mobmastery,omitempty"`    // Tracks particular masteries around a given mob
	Pet                 pets.Pet                       `yaml:"pet,omitempty"`           // Do they have a pet?
	Created             time.Time                      `yaml:"created"`                 // When this character was created
	Timers              map[string]gametime.RoundTimer `yaml:"timers,omitempty"`        // any special timers added to this character
	ZonesVisited        map[string]RoomBitset          `yaml:"zonesvisited,omitempty"`  // permanent record of every room visited, keyed by zone name
	roomHistory         []int                          // A stack FILO of the last X rooms the character has been in
	PlayerDamage        map[int]int                    `yaml:"-"` // key = who, value = how much
	LastPlayerDamage    uint64                         `yaml:"-"` // last round a player damaged this character
	KillerMobInstanceId int                            `yaml:"-"` // transient: mob instance that delivered the killing blow
	KillerMobIsElite    bool                           `yaml:"-"` // transient: true if the killing mob was elite
	KillerMobName       string                         `yaml:"-"` // transient: name of the mob that delivered the killing blow
	permaBuffIds        []int                          // Buff Id's that are always present for this character
	userId              int                            // User ID of the character if any
}

func New() *Character {
	return &Character{
		//Name:   defaultName,
		Adjectives: []string{},
		RoomId:     StartingRoomId,
		Zone:       startingZone,
		RaceId:     startingRace,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 1},
			Speed:      stats.StatInfo{Base: 1},
			Smarts:     stats.StatInfo{Base: 1},
			Vitality:   stats.StatInfo{Base: 1},
			Mysticism:  stats.StatInfo{Base: 1},
			Perception: stats.StatInfo{Base: 1},
		},
		Level:          1,
		Experience:     1,
		TrainingPoints: 0,
		StatPoints:     0,
		TNLScale:       1.0,
		Health:         startingHealth,
		HealthMax:      stats.StatInfo{Base: 1},
		Mana:           startingMana,
		ManaMax:        stats.StatInfo{Base: 1},
		Skills:         make(map[string]int),
		Gold:           25,
		Bank:           100,
		SpellBook:      make(map[string]int),
		CharmedMobs:    []int{},
		Items:          []items.Item{},
		Buffs:          buffs.New(),
		Equipment:      Worn{},
		MiscData:       make(map[string]any),
		roomHistory:    make([]int, 0, 10),
		KeyRing:        make(map[string]string),
		Created:        time.Now(),
		PlayerDamage:   map[int]int{},
		Timers:         map[string]gametime.RoundTimer{},
	}
}

// returns description unless description is a hash
// which points to another description location.
func (c *Character) GetDescription() string {
	if c.FormRaceId > 0 {
		trueRace := races.GetRace(c.RaceId)
		formRace := races.GetRace(c.FormRaceId)
		if trueRace != nil && formRace != nil {
			return strings.ReplaceAll(
				strings.ReplaceAll(c.Description, trueRace.Name, formRace.Name),
				strings.ToLower(trueRace.Name), strings.ToLower(formRace.Name),
			)
		}
	}
	return c.Description
}

// returns description unless description is a hash
// which points to another description location.
func (c *Character) TrackPlayerDamage(userId int, damageAmt int) {

	roundNow := util.GetRoundCount()
	if len(c.PlayerDamage) == 0 {
		c.PlayerDamage = map[int]int{}
	} else {
		if roundNow-c.LastPlayerDamage > 30 {
			clear(c.PlayerDamage)
		}
	}

	c.PlayerDamage[userId] = c.PlayerDamage[userId] + damageAmt
	c.LastPlayerDamage = roundNow

}

/*
All spells should have a 10% minimum chance of success.
*/
func (c *Character) GetBaseCastSuccessChance(spellId string) int {

	sp := spells.GetSpell(spellId)
	if sp == nil {
		return -1
	}

	// start with 100% chance of success
	targetNumber := 100

	// subtract spell difficulty
	// 1-100
	targetNumber -= sp.GetDifficulty()

	// add spell level bonus
	// 10-30
	skillLevel := c.GetSkillLevel(skills.Cast)
	//targetNumber += (skillLevel * 5)
	//targetNumber -= 5 // cancel out the first level

	// add the proficiency of the spell (more casts == better)
	// 0-20
	profFactor := 1.0
	if skillLevel == 2 {
		profFactor = 1.25 // .25 more than lvl 1
	} else if skillLevel == 3 {
		profFactor = 1.75 // .50 more than lvl 2
	} else if skillLevel == 4 {
		profFactor = 2.50 // .75 more than lvl 3
	}
	casts := c.SpellBook[spellId]
	proficiency := int(math.Floor((float64(casts) / 50 * profFactor))) // after 50 casts proficiency is 1
	if proficiency < 0 {
		proficiency = 0
	} else if proficiency > 20 {
		proficiency = 20
	}
	targetNumber += proficiency

	targetNumber += int(math.Floor(float64(c.Stats.Mysticism.ValueAdj) / 5))

	// add by any stat mods for casting, or casting school
	// 0-xx
	targetNumber += c.StatMod(string(statmods.Casting)) + c.StatMod(string(statmods.CastingPrefix)+string(sp.School))

	if targetNumber < 0 {
		targetNumber = 0
	} else if targetNumber > 100 {
		targetNumber = 100
	}

	return targetNumber
}

func (c *Character) CarryCapacity() int {
	return 5 + c.Stats.Strength.ValueAdj/3
}

func (c *Character) DeductActionPoints(amount int) bool {

	if c.ActionPoints < amount {
		return false
	}
	c.ActionPoints -= amount
	if c.ActionPoints < 0 {
		c.ActionPoints = 0
	}
	return true
}

// Sometimes it's useful for a character to know what user it belongs to.
func (c *Character) SetUserId(userId int) {
	c.userId = userId
}

func (c *Character) SetMiscData(key string, value any) {

	if c.MiscData == nil {
		c.MiscData = make(map[string]any)
	}

	if value == nil {
		delete(c.MiscData, key)
		return
	}
	c.MiscData[key] = value
}

func (c *Character) GetMiscData(key string) any {

	if c.MiscData == nil {
		c.MiscData = make(map[string]any)
	}

	if value, ok := c.MiscData[key]; ok {
		return value
	}
	return nil
}

func (c *Character) GetMiscDataKeys(prefixMatch ...string) []string {

	if c.MiscData == nil {
		c.MiscData = make(map[string]any)
	}

	allKeys := []string{}
	for key := range c.MiscData {
		allKeys = append(allKeys, key)
	}

	if len(prefixMatch) == 0 {
		return allKeys
	}

	retKeys := []string{}
	for _, prefix := range prefixMatch {
		for _, key := range allKeys {
			if finalKey, ok := strings.CutPrefix(key, prefix); ok {
				retKeys = append(retKeys, finalKey)
			}
		}
	}

	return retKeys
}

func (c *Character) FindKeyInBackpack(lockId string) (items.Item, bool) {

	lockId = strings.ToLower(lockId)

	for _, itm := range c.GetAllBackpackItems() {
		itmSpec := itm.GetSpec()
		if itmSpec.Type != items.Key {
			continue
		}

		if itmSpec.KeyLockId == lockId {
			return itm, true
		}
	}

	return items.Item{}, false
}

func (c *Character) HasKey(lockId string, difficulty int) (hasKey bool, hasSequence bool) {

	sequence := util.GetLockSequence(lockId, difficulty, string(configs.GetServerConfig().Seed))

	// Check whether they ahve a key for this lock
	return c.GetKey(`key-`+lockId) != ``, c.GetKey(lockId) == sequence
}

func (c *Character) KeyCount() int {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	return len(c.KeyRing)
}

func (c *Character) GetKey(lockId string) string {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	return c.KeyRing[strings.ToLower(lockId)]
}

func (c *Character) SetKey(lockId string, sequence string) {
	if c.KeyRing == nil {
		c.KeyRing = make(map[string]string)
	}
	if len(sequence) == 0 {
		delete(c.KeyRing, strings.ToLower(lockId))
	} else {
		c.KeyRing[strings.ToLower(lockId)] = strings.ToUpper(sequence)
	}
}

func (c *Character) GetDefaultDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {
	// default racial
	raceInfo := races.GetRace(c.GetRaceId())

	attacks = raceInfo.Damage.Attacks
	dCount = raceInfo.Damage.DiceCount
	dSides = raceInfo.Damage.SideCount
	bonus = raceInfo.Damage.BonusDamage
	buffOnCrit = raceInfo.Damage.CritBuffIds

	dCount += int(math.Floor((float64(c.Stats.Speed.ValueAdj) / 50)))
	dSides += int(math.Floor((float64(c.Stats.Strength.ValueAdj) / 12)))
	bonus += int(math.Floor((float64(c.Stats.Perception.ValueAdj) / 25)))

	if dCount < raceInfo.Damage.DiceCount {
		dCount = raceInfo.Damage.DiceCount
	}
	if dSides < raceInfo.Damage.SideCount {
		dSides = raceInfo.Damage.SideCount
	}

	return attacks, dCount, dSides, bonus, buffOnCrit
}

func (c *Character) GetSpells() map[string]int {
	ret := make(map[string]int)
	maps.Copy(ret, c.SpellBook)
	return ret
}

func (c *Character) HasSpell(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		return intVal > 0
	}
	return false
}

func (c *Character) DisableSpell(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		if intVal > 0 {
			c.SpellBook[spellName] = intVal * -1
		}
	}
	return false
}

func (c *Character) EnableSpell(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		if intVal < 0 {
			c.SpellBook[spellName] = intVal * -1
		}
	}
	return false
}

func (c *Character) TrackSpellCast(spellName string) bool {
	if intVal, ok := c.SpellBook[spellName]; ok {
		if intVal > 0 {
			intVal++
			c.SpellBook[spellName] = intVal
		}
	}
	return false
}

func (c *Character) LearnSpell(spellName string) bool {
	if _, ok := c.SpellBook[spellName]; !ok {
		c.SpellBook[spellName] = 1
		return true
	}
	return false
}

func (c *Character) UnLearnSpell(spellName string) bool {
	if _, ok := c.SpellBook[spellName]; !ok {
		return false
	}
	delete(c.SpellBook, spellName)
	return true
}

func (c *Character) GrantXP(xp int) (actualXP int, xpScale int) {

	if xp == 0 {
		return 0, 100
	}

	preScale := float64(configs.GetGamePlayConfig().XPScale) / 100
	xp = int(math.Round(preScale * float64(xp)))

	xpScale = c.StatMod(string(statmods.XPScale)) + 100

	if xpScale == 100 {
		actualXP = xp
	} else {

		scaleFloat := max(float64(xpScale)/100, 1)

		actualXP = int(float64(xp) * scaleFloat)
	}

	c.Experience += actualXP

	mudlog.Debug(`GrantXP()`, `username`, c.Name, `xp`, xp, `xpscale`, xpScale, `actualXP`, actualXP)

	return actualXP, xpScale
}

func (c *Character) TrackCharmed(mobId int, add bool) {
	for pos, mobInstanceId := range c.CharmedMobs {
		if mobInstanceId == mobId {
			if !add {
				c.CharmedMobs = slices.Delete(c.CharmedMobs, pos, pos+1)
			}
			return
		}
	}
	c.CharmedMobs = append(c.CharmedMobs, mobId)
}

func (c *Character) GetCharmIds() []int {
	return append([]int{}, c.CharmedMobs...)
}

func (c *Character) Charm(userId int, rounds int, expireCommand string) {
	c.SetAdjective(`charmed`, true)
	c.Charmed = NewCharm(userId, rounds, expireCommand)
	if c.Aggro != nil && c.Aggro.UserId == userId {
		c.Aggro = nil
	}
}

func (c *Character) KnowsFirstAid() bool {
	if r := races.GetRace(c.GetRaceId()); r != nil {
		return r.KnowsFirstAid
	}
	return false
}

func (c *Character) GetCharmedUserId() int {
	if c.Charmed != nil {
		return c.Charmed.UserId
	}
	return 0
}

func (c *Character) IsCharmed(userId ...int) bool {

	if c.Charmed == nil {
		return false
	}

	if len(userId) == 0 {
		return c.Charmed != nil
	}

	if c.Charmed == nil {
		return false
	}
	return slices.Contains(userId, c.Charmed.UserId)
}

// Returns userId of whoever had charmed them
func (c *Character) RemoveCharm() int {
	charmUserId := 0
	c.SetAdjective(`charmed`, false)
	if c.Charmed != nil {
		charmUserId = c.Charmed.UserId
		c.Charmed = nil
	}
	return charmUserId
}

func (c *Character) GetRandomItem() (items.Item, bool) {
	if len(c.Items) == 0 {
		return items.Item{}, false
	}
	return c.Items[util.Rand(len(c.Items))], true
}

// USERNAME appears to be <BLANK>
func (c *Character) GetHealthAppearance() string {

	className := util.HealthClass(c.Health, c.HealthMax.Value)
	pct := int(float64(c.Health) / float64(c.HealthMax.Value) * 100)

	if pct < 15 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> looks like they're <ansi fg="%s">about to die!</ansi>`, c.Name, className)
	}

	if pct < 50 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> looks to be in <ansi fg="%s">pretty bad shape.</ansi>`, c.Name, className)
	}

	if pct < 80 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> has some <ansi fg="%s">cuts and bruises.</ansi>`, c.Name, className)
	}

	if pct < 100 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="%s">a few scratches.</ansi>`, c.Name, className)
	}

	return fmt.Sprintf(`<ansi fg="username">%s</ansi> is in <ansi fg="%s">perfect health.</ansi>`, c.Name, className)
}

func (c *Character) GetAllSkillRanks() map[string]int {
	retMap := make(map[string]int)
	maps.Copy(retMap, c.Skills)
	return retMap
}

// Returns an integer representing a % damage reduction
func (c *Character) GetDefense() int {

	reduction := 0
	for _, slot := range AllSlots() {
		reduction += c.Equipment.Get(slot).GetDefense()
	}

	//reduction = int(float64(reduction) / 9)

	// If wearing an offhand item like a shield, defense gets a 50% boost
	// Holdables are not considered "shield" type items.
	// Anything held in the offhand that provides a damage reduction is considered a shield.
	if c.Equipment.Offhand.ItemId != 0 && c.Equipment.Offhand.GetSpec().Type != items.Weapon && c.Equipment.Offhand.GetSpec().DamageReduction > 0 {
		reduction = int(float64(reduction) * 1.5)
	}

	if reduction > 100 {
		reduction = 100
	}

	return reduction
}

func (c *Character) GetMobName(viewingUserId int, renderFlags ...NameRenderFlag) FormattedName {
	return c.getFormattedName(viewingUserId, `mobname`, renderFlags...)
}

func (c *Character) GetPlayerName(viewingUserId int, renderFlags ...NameRenderFlag) FormattedName {
	return c.getFormattedName(viewingUserId, `username`, renderFlags...)
}

func (c *Character) HasAdjective(adj string) bool {
	return slices.Contains(c.Adjectives, adj)
}

func (c *Character) SetAdjective(adj string, addToList bool) {
	if c.Adjectives == nil {
		c.Adjectives = []string{}
	}
	for i, a := range c.Adjectives {
		if a == adj {
			if addToList {
				return
			} else {
				c.Adjectives = slices.Delete(c.Adjectives, i, i+1)
				return
			}
		}
	}
	if addToList {
		c.Adjectives = append(c.Adjectives, adj)
	}
}

func (c *Character) GetAdjectives() []string {

	retAdjectives := []string{}

	// Start dynamic adjectives
	if c.Health < 1 {
		retAdjectives = append(retAdjectives, `downed`)
	}

	if len(c.Shop) > 0 {
		retAdjectives = append(retAdjectives, `shop`)
	}

	if c.HasBuffFlag(buffs.EmitsLight) {
		retAdjectives = append(retAdjectives, `lit`)
	}

	if c.HasBuffFlag(buffs.Hidden) {
		retAdjectives = append(retAdjectives, `hidden`)
	}

	if c.HasBuffFlag(buffs.Poison) {
		retAdjectives = append(retAdjectives, `poisoned`)
	}

	if c.FormRaceId > 0 {
		if r := races.GetRace(c.FormRaceId); r != nil {
			retAdjectives = append(retAdjectives, strings.ToLower(r.Name)+` form`)
		}
	}
	// End dynamic adjectives

	retAdjectives = append(retAdjectives, c.Adjectives...)

	return retAdjectives
}

func (c *Character) getFormattedName(viewingUserId int, uType string, renderFlags ...NameRenderFlag) FormattedName {

	f := FormattedName{
		Name:       c.Name,
		Type:       uType,
		Adjectives: make([]string, 0, len(c.Adjectives)),
	}

	includeHealth := false
	for _, flag := range renderFlags {
		if flag == RenderHealth {
			includeHealth = true
		} else if flag == RenderShortAdjectives {
			f.UseShortAdjectives = true
		}
	}

	// If including health, only do so if not downed, because downed shows as its own adjective.
	if includeHealth && c.Health > 0 {
		pctHealth := int(math.Ceil(float64(c.Health) / float64(c.HealthMax.Value) * 100))
		f.Adjectives = append(f.Adjectives, strconv.Itoa(pctHealth)+`%`)
	}

	f.Adjectives = append(f.Adjectives, c.GetAdjectives()...)

	if c.Health < 1 {
		f.Suffix = `downed`
	} else if c.Aggro != nil && c.Aggro.UserId == viewingUserId {
		f.Suffix = `aggro`
	}

	if c.Pet.Exists() {
		f.PetName = c.Pet.DisplayName()
	}

	return OnGetFormattedName.Fire(f)
}

func (c *Character) PruneCooldowns() {
	if len(c.Cooldowns) == 0 {
		return
	}

	c.Cooldowns.Prune()
}

func (c *Character) GetCooldown(trackingTag string) int {
	if c.Cooldowns == nil {
		c.Cooldowns = make(Cooldowns)
	}
	return c.Cooldowns[trackingTag]
}

func (c *Character) GetAllCooldowns() map[string]int {

	ret := map[string]int{}

	if c.Cooldowns == nil {
		return ret
	}

	maps.Copy(ret, c.Cooldowns)

	return ret
}

func (c *Character) TryCooldown(trackingTag string, cooldownTime string) bool {
	if c.Cooldowns == nil {
		c.Cooldowns = make(Cooldowns)
	}

	return c.Cooldowns.Try(trackingTag, cooldownTime)
}

func (c *Character) SetSetting(settingName string, settingValue string) {
	if c.Settings == nil {
		c.Settings = make(map[string]string)
	}

	if settingValue == "" {
		delete(c.Settings, settingName)
	} else {
		c.Settings[settingName] = settingValue
	}
}

func (c *Character) GetSetting(settingName string) string {
	if c.Settings == nil {
		c.Settings = make(map[string]string)
	}
	if settingValue, ok := c.Settings[settingName]; ok {
		return settingValue
	}
	return ""
}

func (c *Character) StoreItem(i items.Item) bool {
	if i.ItemId < 1 {
		return false
	}

	i.Validate()

	c.Items = append(c.Items, i)

	return true
}

func (c *Character) RemoveItem(i items.Item) bool {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			c.Items = append(c.Items[:j], c.Items[j+1:]...)
			return true
		}
	}
	return false
}

func (c *Character) HandsRequired(i items.Item) int {

	if i.ItemId < 1 {
		return 0
	}

	iSpec := i.GetSpec()

	// Shooting weapnos don't benefit from creature size
	// when determining how many hands they require
	if iSpec.Subtype == items.Shooting {
		return iSpec.Hands
	}

	raceInfo := races.GetRace(c.GetRaceId())
	if raceInfo.Size == races.Large {
		return 1
	}

	if raceInfo.Size == races.Small {
		return iSpec.Hands + 1
	}

	return iSpec.Hands
}

// Copies over an existing item with a new item
// Returns true if successfully replaces an item
func (c *Character) UpdateItem(originalItm items.Item, replacement items.Item) bool {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(originalItm) {
			// If the number of uses remaining has decremented from the original item
			// The item gets destroyed from existence
			if originalItm.Uses >= 1 && replacement.Uses < 1 {
				c.Items = append(c.Items[:j], c.Items[j+1:]...)
			} else {
				c.Items[j] = replacement
			}
			return true
		}
	}
	return false
}

func (c *Character) UseItem(i items.Item) int {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			usesLeft := c.Items[j].Uses
			if usesLeft > 0 {
				usesLeft--
			}
			if usesLeft <= 0 {
				c.Items = append(c.Items[:j], c.Items[j+1:]...)
			} else {
				c.Items[j].Uses = usesLeft
				c.Items[j].LastUsedRound = util.GetRoundCount()
			}

			return usesLeft
		}
	}

	return 0
}

func (c *Character) FindInBackpack(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (c *Character) FindOnBody(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	partialMatch, fullMatch := items.FindMatchIn(itemName, c.Equipment.GetAllItems()...)

	if fullMatch.ItemId != 0 {
		return fullMatch, true
	}

	if partialMatch.ItemId != 0 {
		return partialMatch, true
	}

	return items.Item{}, false
}

func (c *Character) GetSkills() map[string]int {
	skillResults := make(map[string]int)
	for skillName, skillLevel := range c.Skills {
		skillResults[skillName] = skillLevel
	}
	return skillResults
}

func (c *Character) SetSkill(skillName string, level int) {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}
	skillName = strings.ToLower(skillName)

	if level == 0 {
		delete(c.Skills, skillName)
		return
	}

	c.Skills[skillName] = level
}

// Increases the skill training counter and returns the new value
func (c *Character) TrainSkill(skillName string, targetLevel ...int) int {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}

	skillName = strings.ToLower(skillName)

	skillLevel := 0

	if lvl, ok := c.Skills[skillName]; ok {
		skillLevel = lvl
	}

	if len(targetLevel) > 0 {

		if skillLevel < targetLevel[0] {
			skillLevel = targetLevel[0]
		}

	} else if skillLevel < 4 {

		skillLevel++

	}

	c.Skills[skillName] = skillLevel

	return skillLevel
}

// Gets the current value of the skillname provided
func (c *Character) GetSkillLevel(skillName skills.SkillTag) int {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}

	if level, ok := c.Skills[string(skillName)]; ok {
		return level
	}
	return 0
}

func (c *Character) GetSkillLevelCost(currentLevel int) int {
	return currentLevel
}

func (c *Character) GetMaxCharmedCreatures() int {
	lvl := c.GetSkillLevel(skills.Tame)
	return lvl + 1
}

func (c *Character) GetMemoryCapacity() int {
	memCap := c.GetSkillLevel(skills.Map) * c.Stats.Smarts.ValueAdj
	if memCap < 0 {
		memCap = 0
	}
	return memCap + 5
}

func (c *Character) GetMapSprawlCapacity() int {
	sprawlCap := c.GetSkillLevel(skills.Map) + (c.Stats.Smarts.ValueAdj >> 2)
	if sprawlCap < 0 {
		sprawlCap = 0
	}
	return sprawlCap
}

// Remember visiting a room. This may cause to forget an older room if the memory is full.
func (c *Character) RememberRoom(roomId int) {
	mapHistory := c.GetMemoryCapacity()
	if len(c.roomHistory) >= mapHistory*2 {
		// Prune out everything except {mapHistory}-1 items at the end
		c.roomHistory = c.roomHistory[len(c.roomHistory)-(mapHistory-1):]
	}
	c.roomHistory = append(c.roomHistory, roomId)
}

// MarkVisitedRoom permanently records that this character has visited roomId
// in the given zone. Safe to call every time a player enters a room.
// Returns true only if this specific call completed the zone (i.e. every room
// in validRoomIds is now visited). Returns false if the room was already
// visited, if validRoomIds is empty, or if the zone is still incomplete.
func (c *Character) MarkVisitedRoom(roomId int, zone string, validRoomIds map[int]struct{}) bool {
	if c.ZonesVisited == nil {
		c.ZonesVisited = make(map[string]RoomBitset)
	}
	if _, ok := c.ZonesVisited[zone]; !ok {
		c.ZonesVisited[zone] = make(RoomBitset)
	}

	// If the bit was already set this call cannot be the completing visit.
	if c.ZonesVisited[zone].Has(roomId) {
		return false
	}

	c.ZonesVisited[zone].Set(roomId)

	if len(validRoomIds) == 0 {
		return false
	}

	return c.ZonesVisited[zone].IsComplete(validRoomIds)
}

// HasVisitedRoom reports whether this character has ever visited roomId in zone.
func (c *Character) HasVisitedRoom(roomId int, zone string) bool {
	if c.ZonesVisited == nil {
		return false
	}
	bs, ok := c.ZonesVisited[zone]
	if !ok {
		return false
	}
	return bs.Has(roomId)
}

// ZoneVisitProgress returns how many rooms the character has visited in zone
// and the total number of rooms in that zone, allowing callers to compute a
// completion percentage. validRoomIds should come from ZoneConfig.RoomIds.
func (c *Character) ZoneVisitProgress(zone string, validRoomIds map[int]struct{}) (visited int, total int) {
	total = len(validRoomIds)
	if c.ZonesVisited == nil {
		return 0, total
	}
	bs, ok := c.ZonesVisited[zone]
	if !ok {
		return 0, total
	}
	return bs.CountIn(validRoomIds), total
}

// ZoneVisitPercent returns the percentage (0–100) of rooms in zone that the
// character has visited. Returns 0 when the zone has no rooms.
func (c *Character) ZoneVisitPercent(zone string, validRoomIds map[int]struct{}) int {
	visited, total := c.ZoneVisitProgress(zone, validRoomIds)
	if total == 0 {
		return 0
	}
	return int(float64(visited) / float64(total) * 100)
}

func (c *Character) IsQuestDone(questToken string) bool {
	testQuestId, _ := quests.TokenToParts(questToken)
	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	stage := c.QuestProgress[testQuestId]

	return stage == `end`
}

func (c *Character) HasQuest(questToken string) bool {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	testQuestId, testQuestStep := quests.TokenToParts(questToken)

	currentStep, ok := c.QuestProgress[testQuestId]
	if !ok {
		return false
	}

	// If on that step currently, then true
	if currentStep == testQuestStep {
		return true
	}

	currentToken := quests.PartsToToken(testQuestId, currentStep)

	// If the current token comes after the test token then they've already done that quest
	return quests.IsTokenAfter(questToken, currentToken)
}

func (c *Character) GetQuestProgress() map[int]string {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	retMap := make(map[int]string)
	for questId, stepName := range c.QuestProgress {
		retMap[questId] = stepName
	}
	return retMap
}

func (c *Character) GiveQuestToken(questToken string) bool {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	questId, newStep := quests.TokenToParts(questToken)
	currentProgress := c.QuestProgress[questId]

	currentToken := quests.PartsToToken(questId, currentProgress)

	if quests.IsTokenAfter(currentToken, questToken) {
		c.QuestProgress[questId] = newStep
		return true
	}

	return false
}

func (c *Character) ClearQuestToken(questToken string) {

	if c.QuestProgress == nil {
		c.QuestProgress = make(map[int]string)
	}

	questId, _ := quests.TokenToParts(questToken)

	delete(c.QuestProgress, questId)
}

func (c *Character) SetAggroRemote(exitName string, userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {
	c.SetAggro(userId, mobInstanceId, aggroType, roundsWaitTime...)
	c.Aggro.ExitName = exitName
}

func (c *Character) SetAggro(userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {

	var combatAddlWaitRounds int = 0

	if len(roundsWaitTime) > 0 {
		for _, waitAmt := range roundsWaitTime {
			combatAddlWaitRounds += waitAmt
		}
	} else {
		combatAddlWaitRounds = c.Equipment.Weapon.GetSpec().WaitRounds + c.Equipment.Offhand.GetSpec().WaitRounds
	}

	if aggroType == DefaultAttack {
		if c.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			aggroType = Shooting
		}
	}

	c.Aggro = &Aggro{
		UserId:        userId,
		MobInstanceId: mobInstanceId,
		Type:          aggroType,
		RoundsWaiting: combatAddlWaitRounds,
	}

}

func (c *Character) SetCast(roundsWaitTime int, sInfo SpellAggroInfo) {

	c.Aggro = &Aggro{
		Type:          SpellCast,
		RoundsWaiting: roundsWaitTime,
		SpellInfo:     sInfo,
	}

}

func (c *Character) EndAggro() {
	c.Aggro = nil
}

func (c *Character) IsAggro(targetUserId int, targetMobInstanceId int) bool {

	if c.Aggro != nil {

		if c.Aggro.MobInstanceId > 0 && c.Aggro.MobInstanceId == targetMobInstanceId {
			return true
		}

		if c.Aggro.UserId > 0 && c.Aggro.UserId == targetUserId {
			return true
		}

		if c.Aggro.Type == SpellCast {
			if len(c.Aggro.SpellInfo.TargetUserIds) > 0 {
				for _, uId := range c.Aggro.SpellInfo.TargetUserIds {
					if uId == targetUserId {
						return true
					}
				}
			}

			if len(c.Aggro.SpellInfo.TargetMobInstanceIds) > 0 {
				for _, mId := range c.Aggro.SpellInfo.TargetMobInstanceIds {
					if mId == targetMobInstanceId {
						return true
					}
				}
			}
		}

	}
	return false
}

func (c *Character) IsDisabled() bool {
	return c.Health <= 0
}

func (c *Character) HasBuffFlag(buffFlag buffs.Flag) bool {
	return c.Buffs.HasFlag(buffFlag, false)
}

func (c *Character) CancelBuffsWithFlag(buffFlag buffs.Flag) bool {
	if c.Buffs.HasFlag(buffFlag, true) {
		c.Validate(true)
		return true
	}
	return false
}

func (c *Character) HasBuff(buffId int) bool {
	return c.Buffs.HasBuff(buffId)
}

func (c *Character) AddBuff(buffId int, isPermanent bool, triggerCountOverride ...int) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuff(buffId, isPermanent, triggerCountOverride...) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	c.Validate()
	return nil
}

func (c *Character) TrackBuffStarted(buffId int) {
	c.Buffs.Started(buffId)
}

func (c *Character) GetBuffs(buffId ...int) []*buffs.Buff {
	return c.Buffs.GetBuffs(buffId...)
}

func (c *Character) RemoveBuff(buffId int) {
	buffId = int(math.Abs(float64(buffId)))
	c.Buffs.RemoveBuff(buffId)
	c.Validate()
}

func (c *Character) TimerSet(name, period string) {
	if c.Timers == nil {
		c.Timers = map[string]gametime.RoundTimer{}
	}
	c.Timers[name] = gametime.RoundTimer{
		RoundStart: util.GetRoundCount(),
		Period:     period,
	}
}

func (c *Character) TimerExpired(name string) bool {
	if c.Timers == nil {
		return true
	}

	t, ok := c.Timers[name]

	if !ok {
		return true
	}

	if t.Expired() {
		delete(c.Timers, name)
		return true
	}

	return false
}

func (c *Character) TimerExists(name string) bool {
	if c.Timers == nil {
		return false
	}

	_, ok := c.Timers[name]
	return ok
}

func (c *Character) ApplyHealthChange(healthChange int) int {
	oldHealth := c.Health
	newHealth := c.Health + healthChange
	if newHealth < 0 {
		c.CancelBuffsWithFlag(buffs.CancelIfCombat)

		// If they haven't dropped yet, require a drop before going straight to death.
		// Don't allow players to drop under -5 in a single hit.
		if newHealth < -5 && oldHealth > 0 {
			newHealth = -5
		} else if newHealth <= -10 {
			newHealth = -10
		}
	} else if newHealth > c.HealthMax.Value {
		newHealth = c.HealthMax.Value
	}

	c.Health = newHealth

	return newHealth - oldHealth
}

func (c *Character) ApplyManaChange(manaChange int) int {
	oldMana := c.Mana
	c.Mana += manaChange
	if c.Mana < 0 {
		c.Mana = 0
	} else if c.Mana > c.ManaMax.Value {
		c.Mana = c.ManaMax.Value
	}
	return c.Mana - oldMana
}

func (c *Character) BarterPrice(startPrice int) int {
	factor := (float64(c.Stats.Perception.ValueAdj) / 3) / 100 // 100 = 33% discount, 0 = 0% discount, 300 = 100% discount
	if factor > .75 {
		factor = .75
	}
	return int(factor * float64(startPrice))
}

func (c *Character) XPTNL() int {
	return c.XPTL(c.Level)
}

// Amt TNL for a specific level
func (c *Character) XPTL(lvl int) int {
	if lvl < 1 {
		lvl = 1
	}
	cfg := configs.GetProgressionConfig()
	base := float64(cfg.XPBase)
	xp := (base + math.Pow(float64(lvl), float64(cfg.XPLevelPower))*float64(cfg.XPLevelFactor)*base) * float64(c.TNLScale)
	if xp > math.MaxInt64 {
		return math.MaxInt64
	}
	return int(xp)
}

// Returns the actual xp in regards to the current level/next level
func (c *Character) XPTNLActual() (xpPastCurrentLevel int, tnlXP int) {

	xpForCurrentLevel := c.XPTL(c.Level - 1)
	if c.Level == 1 {
		xpForCurrentLevel = 0
	}

	xpForNextLevel := c.XPTL(c.Level)
	tnlXP = xpForNextLevel - xpForCurrentLevel

	xpPastCurrentLevel = c.Experience - xpForCurrentLevel

	return xpPastCurrentLevel, tnlXP
}

func (c *Character) LevelUp() (bool, stats.Statistics) {

	if c.XPTNL() > c.Experience {
		return false, stats.Statistics{}
	}

	var statsBefore stats.Statistics = c.Stats

	c.Level++

	cfgProg := configs.GetProgressionConfig()
	if int(cfgProg.TrainingPointsEveryNLevels) <= 1 || c.Level%int(cfgProg.TrainingPointsEveryNLevels) == 0 {
		c.TrainingPoints += int(cfgProg.TrainingPointsPerLevel)
	}
	if int(cfgProg.StatPointsEveryNLevels) <= 1 || c.Level%int(cfgProg.StatPointsEveryNLevels) == 0 {
		c.StatPoints += int(cfgProg.StatPointsPerLevel)
	}

	c.Validate()

	var statsDelta stats.Statistics = c.Stats

	statsDelta.Strength.Value -= statsBefore.Strength.Value
	statsDelta.Speed.Value -= statsBefore.Speed.Value
	statsDelta.Smarts.Value -= statsBefore.Smarts.Value
	statsDelta.Vitality.Value -= statsBefore.Vitality.Value
	statsDelta.Mysticism.Value -= statsBefore.Mysticism.Value
	statsDelta.Perception.Value -= statsBefore.Perception.Value

	c.Health = c.HealthMax.Value
	c.Mana = c.ManaMax.Value

	return true, statsDelta
}

func (c *Character) Heal(hp int, mana int) (int, int) {
	startHP := c.Health
	startMP := c.Mana

	c.Health += hp
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}
	c.Mana += hp
	if c.Mana > c.ManaMax.Value {
		c.Mana = c.ManaMax.Value
	}

	return c.Health - startHP, c.Mana - startMP
}

func (c *Character) HealthPerRound() int {
	return 1 + c.StatMod(string(statmods.HealthRecovery))
	/*
		healAmt := math.Round(float64(c.Stats.Vitality.ValueAdj)/8) +
			math.Round(float64(c.Level)/12) +
			1.0

		return int(healAmt)
	*/
}

func (c *Character) ManaPerRound() int {
	return 1 + c.StatMod(string(statmods.ManaRecovery))
	/*
		healAmt := math.Round(float64(c.Stats.Mysticism.ValueAdj)/8) +
			math.Round(float64(c.Level)/12) +
			1.0

		return int(healAmt)
	*/
}

// Where 1000 = a full round
func (c *Character) MovementCost() int {
	modifier := 3                                // by default they should be able to move 3 times per round.
	modifier += int(c.Level / 15)                // Every 15 levels, get an extra movement.
	modifier += int(c.Stats.Speed.ValueAdj / 15) // Every 15 speed, get an extra movement
	return int(1000 / modifier)
}

func (c *Character) StatMod(statName string) int {
	petMod := 0
	if !c.Pet.IsMissing() {
		petMod = c.Pet.StatMod(statName)
	}
	return c.Equipment.StatMod(statName) + c.Buffs.StatMod(statName) + petMod
}

// returns true if something has changed.
func (c *Character) RecalculateStats() {

	// Make sure racial base stats are set
	beforeHealthMax := c.HealthMax
	beforeManaMax := c.ManaMax
	beforeStats := c.Stats

	if trueRaceInfo := races.GetRace(c.RaceId); trueRaceInfo != nil {
		c.TNLScale = trueRaceInfo.TNLScale
		if c.TNLScale == 0 {
			c.TNLScale = 1.0
		}
	}
	if raceInfo := races.GetRace(c.GetRaceId()); raceInfo != nil {
		c.Stats.Strength.Base = raceInfo.Stats.Strength.Base
		c.Stats.Speed.Base = raceInfo.Stats.Speed.Base
		c.Stats.Smarts.Base = raceInfo.Stats.Smarts.Base
		c.Stats.Vitality.Base = raceInfo.Stats.Vitality.Base
		c.Stats.Mysticism.Base = raceInfo.Stats.Mysticism.Base
		c.Stats.Perception.Base = raceInfo.Stats.Perception.Base
	}

	// Add any mods for equipment
	c.Stats.Strength.Mods = c.StatMod(string(statmods.Strength))
	c.Stats.Speed.Mods = c.StatMod(string(statmods.Speed))
	c.Stats.Smarts.Mods = c.StatMod(string(statmods.Smarts))
	c.Stats.Vitality.Mods = c.StatMod(string(statmods.Vitality))
	c.Stats.Mysticism.Mods = c.StatMod(string(statmods.Mysticism))
	c.Stats.Perception.Mods = c.StatMod(string(statmods.Perception))

	// Recalculate stats
	// Stats are basically:
	// level*base + training + mods
	c.Stats.Strength.Recalculate(c.Level)
	c.Stats.Speed.Recalculate(c.Level)
	c.Stats.Smarts.Recalculate(c.Level)
	c.Stats.Vitality.Recalculate(c.Level)
	c.Stats.Mysticism.Recalculate(c.Level)
	c.Stats.Perception.Recalculate(c.Level)

	// Set HP/MP maxes
	// This relies on the above stats so has to be calculated afterwards
	cfgProg := configs.GetProgressionConfig()
	c.HealthMax.NoCap = true
	c.HealthMax.Mods = int(cfgProg.HPBase) +
		c.StatMod(string(statmods.HealthMax)) +
		int(float64(c.Level)*float64(cfgProg.HPPerLevel)) +
		int(float64(c.Stats.Vitality.ValueAdj)*float64(cfgProg.HPPerVitality))

	c.ManaMax.NoCap = true
	c.ManaMax.Mods = int(cfgProg.ManaBase) +
		c.StatMod(string(statmods.ManaMax)) +
		int(float64(c.Level)*float64(cfgProg.ManaPerLevel)) +
		int(float64(c.Stats.Mysticism.ValueAdj)*float64(cfgProg.ManaPerMysticism))

	// Set max action points
	c.ActionPointsMax.Mods = 200 // hard coded for now

	// Recalculate HP/MP stats
	c.HealthMax.Recalculate(c.Level)
	c.ManaMax.Recalculate(c.Level)
	c.ActionPointsMax.Recalculate(c.Level)

	// HP can't max less than 1, MP can't max less than 0
	if c.ManaMax.Value < 0 {
		c.ManaMax.Value = 0
	}
	if c.HealthMax.Value < 1 {
		c.HealthMax.Value = 1
	}
	if c.ActionPointsMax.Value < 50 {
		c.ActionPointsMax.Value = 50
	}

	if c.userId != 0 {
		changed := false
		// return true if something has changed.
		if beforeStats.Strength.ValueAdj != c.Stats.Strength.ValueAdj {
			changed = true
		} else if beforeStats.Speed.ValueAdj != c.Stats.Speed.ValueAdj {
			changed = true
		} else if beforeStats.Smarts.ValueAdj != c.Stats.Smarts.ValueAdj {
			changed = true
		} else if beforeStats.Vitality.ValueAdj != c.Stats.Vitality.ValueAdj {
			changed = true
		} else if beforeStats.Mysticism.ValueAdj != c.Stats.Mysticism.ValueAdj {
			changed = true
		} else if beforeStats.Perception.ValueAdj != c.Stats.Perception.ValueAdj {
			changed = true
		} else if beforeHealthMax != c.HealthMax {
			changed = true
		} else if beforeManaMax != c.ManaMax {
			changed = true
		}

		if changed {
			events.AddToQueue(events.CharacterStatsChanged{UserId: c.userId})
		}
	}

}

// AutoTrain() spends any training points for this character
func (c *Character) AutoTrain() {

	if c.StatPoints < 0 {
		return
	}

	statPtrs := [...]*int{
		&c.Stats.Strength.Training,
		&c.Stats.Speed.Training,
		&c.Stats.Smarts.Training,
		&c.Stats.Vitality.Training,
		&c.Stats.Mysticism.Training,
		&c.Stats.Perception.Training,
	}

	for i := 0; c.StatPoints > 0; i++ {
		*statPtrs[i%len(statPtrs)]++
		c.StatPoints--
	}

	c.Validate()

}

func (c *Character) CanDualWield() bool {
	return c.GetSkillLevel(skills.DualWield) > 0
}

// Returns whether a correction was in order
func (c *Character) Validate(recalcPermaBuffs ...bool) error {

	if len(c.Description) == 0 {
		c.Description = "They seem thoroughly uninteresting."
	}

	if race := races.GetRace(c.RaceId); race == nil {
		c.RaceId = 1
	}

	if c.Created.IsZero() {
		c.Created = time.Now()
	}

	if c.Pet.Exists() {
		c.Pet.Validate()
	}

	if c.SpellBook == nil {
		c.SpellBook = make(map[string]int)
	}

	if c.Zone == "" {
		c.Zone = startingZone
	}

	if c.Name == "" {
		c.Name = defaultName
	}
	if c.Level < 1 {
		c.Level = 1
	}
	if c.Experience < 1 {
		c.Experience = 1
	}

	c.Buffs.Validate()

	// Do a stats recalc based on equipment, race, level, etc.
	c.RecalculateStats()

	// Recalculate health and mana

	if c.Mana > c.ManaMax.Value {
		c.Mana = c.ManaMax.Value
	}
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}

	if c.Health < -10 {
		c.Health = -10
	}

	if c.Mana < 0 {
		c.Mana = 0
	}

	c.Cooldowns.Prune()

	if c.Alignment < AlignmentMinimum {
		c.Alignment = AlignmentMinimum
	}

	if c.Alignment > AlignmentMaximum {
		c.Alignment = AlignmentMaximum
	}

	// Validate possessed/worn items
	// This helps ensure all in-play items have a uid
	for i := range c.Items {
		c.Items[i].Validate()
	}
	for _, slot := range AllSlots() {
		c.Equipment.Get(slot).Validate()
	}
	// Done with validation

	if raceInfo := races.GetRace(c.GetRaceId()); raceInfo != nil {

		c.Equipment.EnableAll()

		// Are there slots that SHOULD be disabled?
		if len(raceInfo.DisabledSlots) > 0 {

			for _, disabledSlot := range raceInfo.DisabledSlots {

				slotType := items.ItemType(disabledSlot)
				slotItem := c.Equipment.Get(slotType)
				if slotItem == nil {
					continue
				}

				var itemFoundInDisabledSlot items.Item = items.ItemDisabledSlot
				if slotItem.ItemId > 0 {
					itemFoundInDisabledSlot = *slotItem
				}
				c.Equipment.Set(slotType, items.ItemDisabledSlot)

				if !itemFoundInDisabledSlot.IsDisabled() {
					c.StoreItem(itemFoundInDisabledSlot)
					mudlog.Debug("Disabled Check", "error", "Item found in disabled slot", "name", itemFoundInDisabledSlot.Name(), "slot", disabledSlot, "character", c.Name)
				}
			}

		}

	}

	if !c.Equipment.Weapon.IsDisabled() && c.Equipment.Weapon.ItemId > 0 {
		weaponHands := c.HandsRequired(c.Equipment.Weapon)
		offhandHands := 0
		if !c.Equipment.Offhand.IsDisabled() && c.Equipment.Offhand.ItemId > 0 {
			offhandHands = c.HandsRequired(c.Equipment.Offhand)
			if offhandHands < 1 {
				offhandHands = 1
			}
		}
		if weaponHands+offhandHands > 2 {
			if offhandHands > 0 {
				c.StoreItem(c.Equipment.Offhand)
				c.Equipment.Offhand = items.Item{}
			}
			if weaponHands > 2 {
				c.StoreItem(c.Equipment.Weapon)
				c.Equipment.Weapon = items.Item{}
			}
		}
	}

	if len(recalcPermaBuffs) > 0 && recalcPermaBuffs[0] {
		c.reapplyPermabuffs()
	}

	return nil
}

func (c *Character) GetRaceId() int {
	if c.FormRaceId > 0 {
		return c.FormRaceId
	}
	return c.RaceId
}

func (c *Character) Race() string {
	if r := races.GetRace(c.GetRaceId()); r != nil {
		return r.Name
	}
	return `Ghostly Spirit`
}

func (c *Character) RaceSize() string {
	if r := races.GetRace(c.GetRaceId()); r != nil {
		return string(r.Size)
	}
	return string(races.Medium)
}

func (c *Character) IsFormChanged() bool {
	return c.FormRaceId > 0
}

func (c *Character) ApplyFormChange(newRaceId int) []items.Item {
	if races.GetRace(newRaceId) == nil {
		return nil
	}

	if c.IsFormChanged() {
		c.RevertFormChange()
	}

	c.FormRaceId = newRaceId
	c.Validate(true)

	return nil
}

func (c *Character) RevertFormChange() []items.Item {
	c.FormRaceId = 0
	c.Validate(true)

	return nil
}

func (c *Character) UpdateAlignment(amt int) {
	if amt == 0 {
		return
	}
	// Resist movement that pushes further from neutral. Movement toward neutral
	// is always unresisted so redemption is never harder than corruption.
	movingAwayFromNeutral := (amt < 0 && c.Alignment < 0) || (amt > 0 && c.Alignment > 0)
	if movingAwayFromNeutral {
		resistance := math.Abs(float64(c.Alignment)) / 100.0
		scaled := float64(amt) * (1.0 - resistance*0.75)
		if math.Abs(scaled) < 1.0 {
			// Probabilistic floor: even at maximum resistance there is a small
			// chance the tick still lands.
			if util.Rand(100) >= int(math.Abs(scaled)*100) {
				return
			}
			amt = int(math.Copysign(1, float64(amt)))
		} else {
			amt = int(math.Round(scaled))
		}
	}
	newAlignment := int(c.Alignment) + amt
	if newAlignment < int(AlignmentMinimum) {
		newAlignment = int(AlignmentMinimum)
	} else if newAlignment > int(AlignmentMaximum) {
		newAlignment = int(AlignmentMaximum)
	}
	c.Alignment = int8(newAlignment)
}

// DecayAlignment drifts alignment one step toward neutral. The amount decayed
// per call scales quadratically with distance from neutral so extreme alignments
// decay faster and are harder to maintain.
func (c *Character) DecayAlignment() {
	if c.Alignment == 0 {
		return
	}
	norm := math.Abs(float64(c.Alignment)) / 100.0
	decay := int(math.Floor(norm*norm*4)) + 1
	if c.Alignment > 0 {
		c.Alignment -= int8(decay)
		if c.Alignment < 0 {
			c.Alignment = 0
		}
	} else {
		c.Alignment += int8(decay)
		if c.Alignment > 0 {
			c.Alignment = 0
		}
	}
}

func (c *Character) AlignmentName() string {
	return AlignmentToString(c.Alignment)
}

func (c *Character) GetAllBackpackItems() []items.Item {
	return append([]items.Item{}, c.Items...)
}

// BestUpgrades returns a map of equipment slot -> best backpack item that
// beats whatever is currently worn in that slot (or fills an empty slot).
// The caller receives only slots where an upgrade exists; worn items that
// already beat every backpack alternative are omitted.
//
// Two-handed weapon / offhand mutual-exclusion is respected: a two-handed
// weapon candidate is skipped when an offhand item is already worn (or
// already chosen as an upgrade), and an offhand candidate is skipped when a
// two-handed weapon is already worn (or already chosen).
func (c *Character) BestUpgrades() map[items.ItemType]items.Item {

	wornItems := map[items.ItemType]items.Item{}
	for _, itm := range c.Equipment.GetAllItems() {
		wornItems[itm.GetSpec().Type] = itm
	}

	// Pass 1: find the highest-Value backpack item for each slot.
	bestBySlot := map[items.ItemType]items.Item{}
	for _, itm := range c.Items {
		itmSpec := itm.GetSpec()
		if itmSpec.Type != items.Weapon && itmSpec.Subtype != items.Wearable {
			continue
		}
		if prev, ok := bestBySlot[itmSpec.Type]; !ok || itmSpec.Value > prev.GetSpec().Value {
			bestBySlot[itmSpec.Type] = itm
		}
	}

	// Pass 2: keep only slots where the best backpack item beats what is worn
	// (or fills an empty, non-disabled slot), then enforce the two-handed /
	// offhand mutual exclusion.
	upgrades := map[items.ItemType]items.Item{}
	for slotType, candidate := range bestBySlot {
		worn, isWorn := wornItems[slotType]
		if isWorn && candidate.GetSpec().Value <= worn.GetSpec().Value {
			continue
		}
		// Skip disabled slots (ItemId == -1 means the race cannot use this slot).
		if slotItem := c.Equipment.Get(slotType); slotItem != nil && slotItem.IsDisabled() {
			continue
		}
		upgrades[slotType] = candidate
	}

	// Resolve two-handed weapon vs offhand conflict.
	// "Effective offhand" = currently worn offhand that is NOT being replaced.
	effectiveOffhand := false
	if _, upgrading := upgrades[items.Offhand]; !upgrading {
		if _, worn := wornItems[items.Offhand]; worn {
			effectiveOffhand = true
		}
	} else {
		effectiveOffhand = true
	}

	if weaponUpgrade, ok := upgrades[items.Weapon]; ok {
		if c.HandsRequired(weaponUpgrade) == 2 && effectiveOffhand {
			delete(upgrades, items.Weapon)
		}
	}

	// "Effective weapon" = currently worn weapon that is NOT being replaced.
	effectiveTwoHanded := false
	if _, upgrading := upgrades[items.Weapon]; !upgrading {
		if wornWeapon, worn := wornItems[items.Weapon]; worn {
			if c.HandsRequired(wornWeapon) == 2 {
				effectiveTwoHanded = true
			}
		}
	} else {
		if c.HandsRequired(upgrades[items.Weapon]) == 2 {
			effectiveTwoHanded = true
		}
	}

	if effectiveTwoHanded {
		delete(upgrades, items.Offhand)
	}

	return upgrades
}

func (c *Character) GetAllWornItems() []items.Item {
	wornItems := []items.Item{}
	for _, slot := range AllSlots() {
		if itm := c.Equipment.Get(slot); itm.ItemId > 0 {
			wornItems = append(wornItems, *itm)
		}
	}
	return wornItems
}

func (c *Character) GetGearValue() int {
	value := 0
	for _, slot := range AllSlots() {
		if itm := c.Equipment.Get(slot); itm.ItemId > 0 {
			value += itm.GetSpec().Value
		}
	}
	return value
}

func (c *Character) Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string) {

	i.Validate()

	spec := i.GetSpec()

	if spec.Type != items.Weapon && spec.Subtype != items.Wearable {
		return returnItems, false, `That item cannot be equipped.`
	}

	iHandsRequired := c.HandsRequired(i)
	if iHandsRequired > 2 {
		return returnItems, false, `That requires too many hands.`
	}

	// are botht he currently equipped weapon and this weapon claws?
	bothMartial := false
	if spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws {
		bothMartial = true
	}

	canDualWield := c.CanDualWield()

	// Weapons can go in either hand.
	// Only do this if this is a 1 handed weapon
	if spec.Type == items.Weapon && iHandsRequired < 2 {

		// If they can dual wield
		if canDualWield || bothMartial {

			// If they have a weapon equippment and it is 1 handed
			if c.Equipment.Weapon.ItemId != 0 && c.HandsRequired(c.Equipment.Weapon) == 1 {
				// If nothing is in their offhand
				if c.Equipment.Offhand.ItemId == 0 {
					// Put it in the offhand.
					//returnItems = append(returnItems, c.Equipment.Offhand)
					c.Equipment.Offhand = i

					c.reapplyPermabuffs()

					return returnItems, true, ``
				}
			}

		}

	}

	// First handle weapon/offhand, since they are special cases
	switch spec.Type {
	case items.Weapon:
		if c.Equipment.Weapon.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't use a weapon.`
		}

		if !c.Equipment.Offhand.IsDisabled() { // Don't allow equipping on a disabled slot
			// If it's a 2 handed weapon, remove whatever is in the offhand
			if iHandsRequired == 2 || !canDualWield && c.Equipment.Offhand.GetSpec().Type == items.Weapon {
				if c.Equipment.Offhand.IsRemoveLocked() && c.Health > 0 {
					return returnItems, false, `Your ` + c.Equipment.Offhand.DisplayName() + ` is bound to you and cannot be removed.`
				}
				returnItems = append(returnItems, c.Equipment.Offhand)
				c.Equipment.Offhand = items.Item{}
			}
		}

		if c.Equipment.Weapon.IsCursed() {
			return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if c.Equipment.Weapon.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is bound to you and cannot be removed.`
		}

		returnItems = append(returnItems, c.Equipment.Weapon)
		c.Equipment.Weapon = i
	case items.Offhand:
		if c.Equipment.Offhand.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't hold things in an offhand.`
		}

		if !c.Equipment.Weapon.IsDisabled() { // Don't allow equipping on a disabled slot
			// If they have a 2h weapon equipped, remove it
			if c.HandsRequired(c.Equipment.Weapon) == 2 {
				// If the weapon is cursed, do not allow the offhand to be equipped
				if c.Equipment.Weapon.IsCursed() {
					return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
				}
				if c.Equipment.Weapon.IsRemoveLocked() && c.Health > 0 {
					return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is bound to you and cannot be removed.`
				}
				returnItems = append(returnItems, c.Equipment.Weapon)
				c.Equipment.Weapon = items.Item{}
			}
		}
		if c.Equipment.Offhand.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Offhand.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Offhand)
		c.Equipment.Offhand = i
	case items.Head:
		if c.Equipment.Head.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your head.`
		}
		if c.Equipment.Head.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Head.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Head)
		c.Equipment.Head = i
	case items.Neck:
		if c.Equipment.Neck.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your neck.`
		}
		if c.Equipment.Neck.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Neck.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Neck)
		c.Equipment.Neck = i
	case items.Body:
		if c.Equipment.Body.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your body.`
		}
		if c.Equipment.Body.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Body.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Body)
		c.Equipment.Body = i
	case items.Belt:
		if c.Equipment.Belt.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your head.`
		}
		if c.Equipment.Belt.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Belt.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Belt)
		c.Equipment.Belt = i
	case items.Gloves:
		if c.Equipment.Gloves.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things as gloves.`
		}
		if c.Equipment.Gloves.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Gloves.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Gloves)
		c.Equipment.Gloves = i
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear rings.`
		}
		if c.Equipment.Ring.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Ring.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Ring)
		c.Equipment.Ring = i
	case items.Legs:
		if c.Equipment.Legs.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your legs.`
		}
		if c.Equipment.Legs.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Legs.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Legs)
		c.Equipment.Legs = i
	case items.Feet:
		if c.Equipment.Feet.IsDisabled() { // Don't allow equipping on a disabled slot
			return returnItems, false, `You can't wear things on your feet.`
		}
		if c.Equipment.Feet.IsRemoveLocked() && c.Health > 0 {
			return returnItems, false, `Your ` + c.Equipment.Feet.DisplayName() + ` is bound to you and cannot be removed.`
		}
		returnItems = append(returnItems, c.Equipment.Feet)
		c.Equipment.Feet = i
	default:
		return returnItems, false, `Unrecognized object.`
	}

	c.reapplyPermabuffs(returnItems...)

	return returnItems, true, ``
}

func (c *Character) RemoveFromBody(i items.Item) bool {

	for _, slot := range AllSlots() {
		if i.Equals(*c.Equipment.Get(slot)) {
			if i.IsRemoveLocked() && c.Health > 0 {
				return false
			}
			c.Equipment.Set(slot, items.Item{})
			c.reapplyPermabuffs(i)
			return true
		}
	}

	return false
}

// Used with SpawnInfo to gift spawning mobs with permabuffs
func (c *Character) SetPermaBuffs(buffIds []int) {
	c.permaBuffIds = buffIds
}

func (c *Character) reapplyPermabuffs(removedItems ...items.Item) {

	buffIdCount := map[int]int{}

	for _, buffId := range c.permaBuffIds {
		buffIdCount[buffId] = 100 // Special case permabuffs associated with certain mobs
	}

	// Apply any buffs that come from a race
	if rInfo := races.GetRace(c.GetRaceId()); rInfo != nil {
		for _, buffId := range rInfo.BuffIds {
			buffIdCount[buffId] = 100 // Don't allow racial buffs to be removed, keep this number high
		}
	}

	// Apply any buffs from pet
	if c.Pet.Exists() && !c.Pet.IsMissing() {
		for _, buffId := range c.Pet.GetBuffs() {
			buffIdCount[buffId] = 100 // Don't allow pet buffs to be removed, keep this number high
		}
	}

	// Track any buffs that come from an item
	// If these don't show up as still being required by an item (such as a yaml file was changed)
	// This will cause them to be removed.
	for _, b := range c.Buffs.List {
		if b.PermaBuff {
			if _, ok := buffIdCount[b.BuffId]; !ok {
				buffIdCount[b.BuffId] = 0
			}
		}
	}

	// Make a list of all item buffs provided by existing worn items
	for _, itm := range c.GetAllWornItems() {
		spec := itm.GetSpec()
		for _, buffId := range spec.WornBuffIds {
			buffIdCount[buffId] = buffIdCount[buffId] + 1
		}

	}
	// Remove any buffs that come specifically from item
	for _, removedItem := range removedItems {
		iSpec := removedItem.GetSpec()
		if len(iSpec.WornBuffIds) > 0 {
			for _, buffId := range iSpec.WornBuffIds {
				buffIdCount[buffId] = buffIdCount[buffId] - 1
			}
		}
	}

	for buffId, ct := range buffIdCount {
		if ct < 1 {
			c.RemoveBuff(buffId)
		} else {
			c.AddBuff(buffId, true)
		}
	}
}

func (c *Character) Uncurse() []items.Item {

	uncursedList := []items.Item{}

	for _, slot := range AllSlots() {
		itm := c.Equipment.Get(slot)
		if itm.IsCursed() {
			itm.Uncursed = true
			uncursedList = append(uncursedList, *itm)
		}
	}

	return uncursedList
}
