package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Equip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "all" {
		return Gearup(``, user, room, flags)
	}

	if rest == "" {
		user.SendText(`Wear WHAT?`)
		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to wear.`, rest))
	} else {

		iSpec := matchItem.GetSpec()
		if iSpec.Type != items.Weapon && iSpec.Subtype != items.Wearable {
			user.SendText(
				fmt.Sprintf(`Your <ansi fg="item">%s</ansi> doesn't look very fashionable.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		// Snapshot stats before equipping
		statsBefore := captureStatSnapshot(user.Character)

		// Swap the item location
		oldItems, wearSuccess, failureReason := user.Character.Wear(matchItem)

		if wearSuccess {

			user.Character.CancelBuffsWithFlag(buffs.Hidden)

			user.Character.RemoveItem(matchItem)

			for _, oldItem := range oldItems {
				if oldItem.ItemId != 0 {
					user.SendText(
						fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, oldItem.DisplayName()),
					)
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, user.Character.Name, oldItem.DisplayName()),
						user.UserId,
					)

					user.Character.StoreItem(oldItem)
				}
			}

			// Trigger any outstanding buff onStart events
			if len(matchItem.GetSpec().WornBuffIds) > 0 {
				for _, buff := range user.Character.Buffs.List {
					if buff.OnStartWaiting {
						if _, err := scripting.TryBuffScriptEvent(`onStart`, user.UserId, 0, buff.BuffId); err == nil {
							user.Character.TrackBuffStarted(buff.BuffId)
						}
					}

				}
			}

			user.Character.Validate(true)

			statDiff := formatStatChanges(statsBefore, captureStatSnapshot(user.Character))

			if iSpec.Subtype == items.Wearable {
				user.SendText(
					fmt.Sprintf(`You wear your <ansi fg="item">%s</ansi>.%s`, matchItem.DisplayName(), statDiff),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> puts on their <ansi fg="item">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()),
					user.UserId,
				)
			} else {
				user.SendText(
					fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi>. You're feeling dangerous.%s`, matchItem.DisplayName(), statDiff),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> wields their <ansi fg="item">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()),
					user.UserId,
				)
			}

			events.AddToQueue(events.EquipmentChange{
				UserId:       user.UserId,
				ItemsWorn:    []items.Item{matchItem},
				ItemsRemoved: oldItems,
			})

		} else {
			if len(failureReason) == 1 {
				failureReason = fmt.Sprintf(`You can't figure out how to equip the <ansi fg="item">%s</ansi>.`, matchItem.DisplayName())
			}
			user.SendText(
				failureReason,
			)
		}

	}

	return true, nil
}

type statSnapshot struct {
	Strength   int
	Speed      int
	Smarts     int
	Vitality   int
	Mysticism  int
	Perception int
	HealthMax  int
	ManaMax    int
}

func captureStatSnapshot(c *characters.Character) statSnapshot {
	return statSnapshot{
		Strength:   c.Stats.Strength.ValueAdj,
		Speed:      c.Stats.Speed.ValueAdj,
		Smarts:     c.Stats.Smarts.ValueAdj,
		Vitality:   c.Stats.Vitality.ValueAdj,
		Mysticism:  c.Stats.Mysticism.ValueAdj,
		Perception: c.Stats.Perception.ValueAdj,
		HealthMax:  c.HealthMax.Value,
		ManaMax:    c.ManaMax.Value,
	}
}

func formatStatChanges(before, after statSnapshot) string {
	type statChange struct {
		name  string
		delta int
	}
	changes := []statChange{
		{"strength", after.Strength - before.Strength},
		{"speed", after.Speed - before.Speed},
		{"smarts", after.Smarts - before.Smarts},
		{"vitality", after.Vitality - before.Vitality},
		{"mysticism", after.Mysticism - before.Mysticism},
		{"perception", after.Perception - before.Perception},
		{"health", after.HealthMax - before.HealthMax},
		{"mana", after.ManaMax - before.ManaMax},
	}

	var parts []string
	for _, c := range changes {
		if c.delta == 0 {
			continue
		}
		var color string
		var sign string
		if c.delta > 0 {
			color = "157"
			sign = "+"
		} else {
			color = "204"
			sign = ""
		}
		parts = append(parts, fmt.Sprintf(`<ansi fg="%s">%s%d %s</ansi>`, color, sign, c.delta, c.name))
	}

	if len(parts) == 0 {
		return ""
	}
	return " ( " + strings.Join(parts, ", ") + " )"
}
