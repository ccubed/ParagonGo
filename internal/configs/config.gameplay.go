package configs

import (
	"strconv"
	"strings"
)

type GamePlay struct {
	AllowItemBuffRemoval ConfigBool `yaml:"AllowItemBuffRemoval"`
	// Death related settings
	Death GameplayDeath `yaml:"Death"`
	// Party settings
	Party GameplayParty `yaml:"Party"`

	LivesStart     ConfigInt `yaml:"LivesStart"`     // Starting permadeath lives
	LivesMax       ConfigInt `yaml:"LivesMax"`       // Maximum permadeath lives
	LivesOnLevelUp ConfigInt `yaml:"LivesOnLevelUp"` // # lives gained on level up
	PricePerLife   ConfigInt `yaml:"PricePerLife"`   // Price in gold to buy new lives
	// Shops/Conatiners
	ShopRestockRate  ConfigString `yaml:"ShopRestockRate"`  // Default time it takes to restock 1 quantity in shops
	ContainerSizeMax ConfigInt    `yaml:"ContainerSizeMax"` // How many objects containers can hold before overflowing
	Combat           CombatConfig `yaml:"Combat"`

	// PVP Restrictions
	PVP GameplayPVP `yaml:"PVP"`
	// XpScale (difficulty)
	XPScale           ConfigFloat `yaml:"XPScale"`
	MobConverseChance ConfigInt   `yaml:"MobConverseChance"` // Chance 1-100 of attempting to converse when idle
}

// CombatConfig holds configurable min/max bounds for every combat calculation.
type CombatConfig struct {
	ConsistentAttackMessages ConfigBool `yaml:"ConsistentAttackMessages"` // Whether each weapon has consistent attack messages

	// Damage bonus (Strength delta drives this)
	DamageBonusMin ConfigInt `yaml:"DamageBonusMin"` // Minimum flat damage bonus
	DamageBonusMax ConfigInt `yaml:"DamageBonusMax"` // Maximum flat damage bonus

	// Chance to hit (Speed delta drives this)
	ToHitMin ConfigInt `yaml:"ToHitMin"` // Minimum hit chance (percent, 0-100)
	ToHitMax ConfigInt `yaml:"ToHitMax"` // Maximum hit chance (percent, 0-100)

	// Extra attacks - weaponless/claws only (Speed delta drives this)
	ExtraAttacksMin ConfigInt `yaml:"ExtraAttacksMin"` // Minimum extra attacks
	ExtraAttacksMax ConfigInt `yaml:"ExtraAttacksMax"` // Maximum extra attacks

	// Chance to crit (Smarts delta drives this)
	CritChanceMin ConfigInt `yaml:"CritChanceMin"` // Minimum crit chance (percent, 0-100)
	CritChanceMax ConfigInt `yaml:"CritChanceMax"` // Maximum crit chance (percent, 0-100)

	// Crit damage multiplier (Perception delta drives this)
	CritMultMin ConfigFloat `yaml:"CritMultMin"` // Minimum crit damage multiplier
	CritMultMax ConfigFloat `yaml:"CritMultMax"` // Maximum crit damage multiplier

	// Chance to dodge (Perception delta drives this)
	DodgeChanceMin ConfigInt `yaml:"DodgeChanceMin"` // Minimum dodge chance (percent, 0-100)
	DodgeChanceMax ConfigInt `yaml:"DodgeChanceMax"` // Maximum dodge chance (percent, 0-100)
}

type GameplayParty struct {
	MaxPlayerCount ConfigInt  `yaml:"MaxPlayerCount"` // Maximum number of players allowed in a party (0 = unlimited)
	SameRoomOnly   ConfigBool `yaml:"SameRoomOnly"`   // Whether players must be in the same room to create/invite/join parties
}

type GameplayPVP struct {
	Enabled      ConfigString `yaml:"Enabled"`      // Possible values: enabled, disabled, limited
	MinimumLevel ConfigInt    `yaml:"MinimumLevel"` // Minimum level required to participate in PVP
}

type GameplayDeath struct {
	EquipmentDropChance ConfigFloat  `yaml:"EquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	AlwaysDropBackpack  ConfigBool   `yaml:"AlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	XPPenalty           ConfigString `yaml:"XPPenalty"`           // Possible values are: none, level, 10%, 25%, 50%, 75%, 90%, 100%
	ProtectionLevels    ConfigInt    `yaml:"ProtectionLevels"`    // How many levels is the user protected from death penalties for?
	PermaDeath          ConfigBool   `yaml:"PermaDeath"`          // Is permadeath enabled?
	CorpsesEnabled      ConfigBool   `yaml:"CorpsesEnabled"`      // Whether corpses are left behind after mob/player deaths
	CorpseDecayTime     ConfigString `yaml:"CorpseDecayTime"`     // How long until corpses decay to dust (go away)
}

func (g *GamePlay) Validate() {

	if g.Party.MaxPlayerCount < 0 {
		g.Party.MaxPlayerCount = 0
	}

	// Ignore AllowItemBuffRemoval
	// Ignore OnDeathAlwaysDropBackpack
	// Ignore CorpsesEnabled

	if g.Death.EquipmentDropChance < 0.0 || g.Death.EquipmentDropChance > 1.0 {
		g.Death.EquipmentDropChance = 0.0 // default
	}

	g.Death.XPPenalty.Set(strings.ToLower(string(g.Death.XPPenalty)))

	if g.Death.XPPenalty != `none` && g.Death.XPPenalty != `level` {
		// If not a valid percent, set to default
		if !strings.HasSuffix(string(g.Death.XPPenalty), `%`) {
			g.Death.XPPenalty = `none` // default
		} else {
			// If not a valid percent, set to default
			percent, err := strconv.ParseInt(string(g.Death.XPPenalty)[0:len(g.Death.XPPenalty)-1], 10, 64)
			if err != nil || percent < 0 || percent > 100 {
				g.Death.XPPenalty = `none` // default
			}
		}
	}

	if g.Death.ProtectionLevels < 0 {
		g.Death.ProtectionLevels = 0 // default
	}

	if g.LivesStart < 0 {
		g.LivesStart = 0
	}

	if g.LivesMax < 0 {
		g.LivesMax = 0
	}

	if g.LivesOnLevelUp < 0 {
		g.LivesOnLevelUp = 0
	}

	if g.PricePerLife < 1 {
		g.PricePerLife = 1
	}

	if g.ShopRestockRate == `` {
		g.ShopRestockRate = `6 hours`
	}

	if g.ContainerSizeMax < 1 {
		g.ContainerSizeMax = 1
	}

	if g.Death.CorpseDecayTime == `` {
		g.Death.CorpseDecayTime = `1 hour`
	}

	if g.PVP.Enabled != PVPEnabled && g.PVP.Enabled != PVPDisabled && g.PVP.Enabled != PVPLimited {
		if g.PVP.Enabled == PVPOff {
			g.PVP.Enabled = PVPDisabled
		} else {
			g.PVP.Enabled = PVPEnabled
		}
	}

	if int(g.PVP.MinimumLevel) < 0 {
		g.PVP.MinimumLevel = 0
	}

	if g.XPScale <= 0 {
		g.XPScale = 100
	}

	if g.MobConverseChance < 0 {
		g.MobConverseChance = 0
	} else if g.MobConverseChance > 100 {
		g.MobConverseChance = 100
	}

	g.Combat.validate()

}

func (c *CombatConfig) validate() {
	// Damage bonus
	if c.DamageBonusMax < 1 {
		c.DamageBonusMax = 10
	}
	if c.DamageBonusMin < 0 {
		c.DamageBonusMin = 0
	}
	if c.DamageBonusMin > c.DamageBonusMax {
		c.DamageBonusMin = 0
	}

	// To-hit
	if c.ToHitMax < 1 || c.ToHitMax > 100 {
		c.ToHitMax = 100
	}
	if c.ToHitMin < 1 {
		c.ToHitMin = 25
	}
	if c.ToHitMin > c.ToHitMax {
		c.ToHitMin = 25
	}

	// Extra attacks (weaponless/claws)
	if c.ExtraAttacksMax == 0 {
		c.ExtraAttacksMax = 3
	} else if c.ExtraAttacksMax < 0 {
		c.ExtraAttacksMax = 0
	}
	if c.ExtraAttacksMin < 0 {
		c.ExtraAttacksMin = 0
	}
	if c.ExtraAttacksMin > c.ExtraAttacksMax {
		c.ExtraAttacksMin = 0
	}

	// Crit chance
	if c.CritChanceMax < 1 || c.CritChanceMax > 100 {
		c.CritChanceMax = 30
	}
	if c.CritChanceMin < 1 {
		c.CritChanceMin = 5
	}
	if c.CritChanceMin > c.CritChanceMax {
		c.CritChanceMin = 5
	}

	// Crit multiplier
	if c.CritMultMin < 1.0 {
		c.CritMultMin = 1.5
	}
	if c.CritMultMax < c.CritMultMin {
		c.CritMultMax = 3.0
	}

	// Dodge chance
	if c.DodgeChanceMax < 1 || c.DodgeChanceMax > 100 {
		c.DodgeChanceMax = 30
	}
	if c.DodgeChanceMin < 1 {
		c.DodgeChanceMin = 5
	}
	if c.DodgeChanceMin > c.DodgeChanceMax {
		c.DodgeChanceMin = 5
	}
}

func GetGamePlayConfig() GamePlay {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay
}

func GetPVPConfig() GameplayPVP {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay.PVP
}

func GetCombatConfig() CombatConfig {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay.Combat
}
