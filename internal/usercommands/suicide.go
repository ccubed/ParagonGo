package usercommands

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Suicide(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	config := configs.GetGamePlayConfig()
	currentRound := util.GetRoundCount()

	if user.Character.Zone == `Shadow Realm` {
		user.SendText(`You're already dead!`)
		return true, errors.New(`already dead`)
	}

	// Give the global user script a chance to override death entirely (e.g. a
	// phoenix-style revival). If it returns true, the death is aborted and the
	// script is responsible for whatever happens instead.
	if handled, _ := scripting.TryUserDieEvent(user.UserId); handled {
		return true, nil
	}

	// Send a death msg to everyone in the room.
	room.SendText(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> has died.`, user.Character.Name),
		user.UserId,
	)

	i := 0
	dmgCt := len(user.Character.PlayerDamage)

	if dmgCt > 0 {
		user.Character.KD.AddPvpDeath()
	} else {
		user.Character.KD.AddMobDeath()
	}

	killedByUserIds := []int{}
	killedBy := ``
	for uid, _ := range user.Character.PlayerDamage {

		if u := users.GetByUserId(uid); u != nil {

			// Update PK stats
			user.Character.KD.AddPlayerDeath(u.UserId, u.Character.Name)
			u.Character.KD.AddPlayerKill(user.UserId, user.Character.Name)

			if i > 0 {
				if i < dmgCt-1 {
					killedBy += `, `
				} else {
					killedBy += ` and `
				}
			}
			killedBy += `<ansi fg="username">` + u.Character.Name + `</ansi>`
			i++
		}

		killedByUserIds = append(killedByUserIds, uid)
	}

	msg := fmt.Sprintf(`<ansi fg="magenta-bold">***</ansi> <ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED!</ansi> <ansi fg="magenta-bold">***</ansi>%s`, user.Character.Name, term.CRLFStr)
	if killedBy != `` {
		msg = fmt.Sprintf(`<ansi fg="magenta-bold">***</ansi> <ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED!</ansi> (killed by %s) <ansi fg="magenta-bold">***</ansi>%s`, user.Character.Name, killedBy, term.CRLFStr)
	}

	events.AddToQueue(events.Broadcast{
		Text: msg,
	})

	allowPenalties := user.Character.Level > int(config.Death.ProtectionLevels)

	killerMobId := 0
	if user.Character.KillerMobInstanceId > 0 {
		if killerMob := mobs.GetInstance(user.Character.KillerMobInstanceId); killerMob != nil {
			killerMobId = int(killerMob.MobId)
		}
		if user.Character.KillerMobIsElite {
			user.Character.KD.AddEliteDeath(killerMobId, user.Character.KillerMobName)
		}
		user.Character.KillerMobInstanceId = 0
		user.Character.KillerMobIsElite = false
		user.Character.KillerMobName = ``
	}

	events.AddToQueue(events.PlayerDeath{
		UserId:        user.UserId,
		RoomId:        user.Character.RoomId,
		Username:      user.Username,
		CharacterName: user.Character.Name,
		Permanent:     allowPenalties && bool(config.Death.PermaDeath) && user.Character.ExtraLives == 0,
		KilledByUsers: killedByUserIds,
		KillerMobId:   killerMobId,
	})

	// If permadeath is enabled, do some extra bookkeeping
	if allowPenalties && bool(config.Death.PermaDeath) {

		if user.Character.ExtraLives > 0 {

			user.Character.ExtraLives--

		} else {

			user.EventLog.Add(`death`, fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="red-bold">PERMA-DIED</ansi>`, user.Character.Name))

			// Perma-died!!!
			textOut, _ := templates.Process("character/permadeath", nil, user.UserId)
			user.SendText(colorpatterns.ApplyColorPattern(textOut, `red`))

			// Unequip everything
			for _, itm := range user.Character.GetAllWornItems() {
				if itm.IsRemoveLocked() {
					continue
				}
				Remove(itm.Name(), user, room, flags)
			}
			// drop all items / gold
			Drop("all", user, room, flags)

			rooms.MoveToRoom(user.UserId, -1)

			user.Character = characters.New()

			return true, nil
		}

	}

	user.EventLog.Add(`death`, fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="red-bold">DIED</ansi>`, user.Character.Name))

	// Only apply penalties if they were above the threshold
	if allowPenalties && !user.Character.HasBuffFlag(buffs.PermaGear) {

		corpseItems := []items.Item{}
		corpseGold := 0

		if config.Death.EquipmentDropChance >= 0 {
			chanceInt := int(config.Death.EquipmentDropChance * 100)
			for _, itm := range user.Character.GetAllWornItems() {
				if itm.IsRemoveLocked() {
					continue
				}
				if util.Rand(100) < chanceInt {

					Remove(itm.Name(), user, room, flags)

					if config.Death.CorpseItems && config.Death.CorpsesEnabled {
						// Item was removed from equipment slot; grab it from backpack and hold for corpse
						if held, found := user.Character.FindInBackpack(itm.Name()); found {
							user.Character.RemoveItem(held)
							corpseItems = append(corpseItems, held)
						}
					} else {
						Drop(itm.Name(), user, room, flags)
					}

				}
			}
		}

		if user.Character.Gold > 0 {
			if config.Death.CorpseItems && config.Death.CorpsesEnabled {
				corpseGold = user.Character.Gold
				user.Character.Gold = 0
			} else {
				user.EventLog.Add(`death`, fmt.Sprintf(`Dropped <ansi fg="gold">%d gold</ansi> on death`, user.Character.Gold))
				Drop(fmt.Sprintf(`%d gold`, user.Character.Gold), user, room, flags)
			}
		}

		if config.Death.AlwaysDropBackpack {
			if config.Death.CorpseItems && config.Death.CorpsesEnabled {
				for _, itm := range user.Character.GetAllBackpackItems() {
					user.Character.RemoveItem(itm)
					corpseItems = append(corpseItems, itm)
				}
			} else {
				Drop("all", user, room, flags)
				user.EventLog.Add(`death`, `Dropped <ansi fg="alert-3">everthing in your backpack</ansi> on death`)
			}
		} else if config.Death.EquipmentDropChance >= 0 {
			chanceInt := int(config.Death.EquipmentDropChance * 100)
			for _, itm := range user.Character.GetAllBackpackItems() {
				if util.Rand(100) < chanceInt {
					if config.Death.CorpseItems && config.Death.CorpsesEnabled {
						user.Character.RemoveItem(itm)
						corpseItems = append(corpseItems, itm)
					} else {
						Drop(itm.Name(), user, room, flags)
						user.EventLog.Add(`death`, fmt.Sprintf(`Dropped your <ansi fg="itemname">%s</ansi> on death`, itm.Name()))
					}
				}
			}
		}

		if user.Character.Level > 1 {

			if config.Death.XPPenalty != `none` {

				if config.Death.XPPenalty == `level` { // are they being brought down to the base of their current level?
					user.Character.Level--
					oldExperience := user.Character.Experience
					user.Character.Experience = user.Character.XPTNL()
					user.Character.Level++

					user.SendText(fmt.Sprintf(`You lost <ansi fg="yellow">%d experience points</ansi>.`, oldExperience-user.Character.Experience))

					user.EventLog.Add(`death`, fmt.Sprintf(`Lost <ansi fg="yellow">%d experience points</ansi> on death`, oldExperience-user.Character.Experience))

				} else {

					var pct float64 = 0.0

					percent, err := strconv.ParseInt(string(config.Death.XPPenalty)[0:len(config.Death.XPPenalty)-1], 10, 64)
					if err == nil && percent >= 0 && percent <= 100 {
						pct = float64(percent) / 100.0
					}

					loss := int(math.Floor(float64(user.Character.Experience) * pct))
					user.Character.Experience -= loss

					user.SendText(fmt.Sprintf(`You lost <ansi fg="yellow">%d experience points</ansi>.`, loss))

					user.EventLog.Add(`death`, fmt.Sprintf(`Lost <ansi fg="yellow">%d experience points</ansi> on death`, loss))
				}
			}

		}

		if config.Death.CorpsesEnabled {
			c := rooms.Corpse{
				UserId:       user.UserId,
				Character:    *user.Character,
				RoundCreated: currentRound,
			}
			if config.Death.CorpseItems {
				c.Items = corpseItems
				c.Gold = corpseGold
			}
			room.AddCorpse(c)
		}

	} else {

		if config.Death.CorpsesEnabled {
			room.AddCorpse(rooms.Corpse{
				UserId:       user.UserId,
				Character:    *user.Character,
				RoundCreated: currentRound,
			})
		}

	}

	user.Character.CancelBuffsWithFlag(buffs.All)

	user.Character.Health = -10
	user.Character.Mana = 0
	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

	clear(user.Character.PlayerDamage)

	rooms.MoveToRoom(user.UserId, int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom))

	return true, nil
}
