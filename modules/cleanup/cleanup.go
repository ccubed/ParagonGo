package cleanup

import (
	"embed"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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
	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	c := CleanupModule{
		plug: plugins.New(`cleanup`, `1.0`),
	}

	//
	// Add the embedded filesystem
	//
	if err := c.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}

	c.plug.Web.AdminPage("Config", "cleanup-config", "html/admin/cleanup-config.html", true, "Modules", "Cleanup", "Configure automated world cleanup schedules and rules.", "Automated cleanup module removing stale corpses, items, and temporary objects.", nil)
	c.plug.Web.AdminPage("About", "cleanup-about", "html/admin/cleanup-about.html", true, "Modules", "Cleanup", "Information and version details for the Cleanup module.", "", nil)
	//
	// Register any user/mob commands
	//
	c.plug.AddUserCommand("trash", c.userTrashCommand, false, false)
	c.plug.AddUserCommand("bury", c.userBuryCommand, false, false)

	c.plug.AddMobCommand("trash", c.mobTrashCommand, false)
	c.plug.AddMobCommand("bury", c.mobBuryCommand, false)

}

// Using a struct gives a way to store longer term data.
type CleanupModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin

	DefaultTrashExperienceValue int
}

func (c *CleanupModule) loadConfig() {

	if experienceValue, ok := c.plug.Config.Get("ExperienceValue").(int); ok {
		c.DefaultTrashExperienceValue = experienceValue
	}

}

func (c *CleanupModule) userTrashCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to trash.`, rest))
	} else {

		c.loadConfig()

		isSneaking := user.Character.HasBuffFlag(buffs.Hidden)

		user.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   matchItem,
			Gained: false,
		})

		user.SendText(
			fmt.Sprintf(`You trash the <ansi fg="item">%s</ansi> for good.`, matchItem.DisplayName()))

		if !isSneaking {
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> destroys <ansi fg="item">%s</ansi>...`, user.Character.Name, matchItem.DisplayName()),
				user.UserId)
		}

		if c.DefaultTrashExperienceValue > 0 {

			// Grant experience equal to a tenth of the item value
			iSpec := matchItem.GetSpec()

			xpGrant := int(float64(iSpec.Value) / 10)
			if xpGrant < c.DefaultTrashExperienceValue {
				xpGrant = c.DefaultTrashExperienceValue
			}
			user.GrantXP(xpGrant, `trash cleanup`)

		}

	}

	return true, nil
}

func (c *CleanupModule) mobTrashCommand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Check whether the user has an item in their inventory that matches
	matchItem, found := mob.Character.FindInBackpack(rest)

	if found {
		mob.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: mob.InstanceId,
			Item:          matchItem,
			Gained:        false,
		})

	}

	return true, nil
}

func (c *CleanupModule) userBuryCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText("Bury what?")
		return true, nil
	}

	if corpse, corpseFound := room.FindCorpse(rest); corpseFound {

		if room.RemoveCorpse(corpse) {

			corpseColor := `mob-corpse`
			if corpse.UserId > 0 {
				corpseColor = `user-corpse`
			}

			user.SendText(fmt.Sprintf(`You bury the <ansi fg="%s">%s corpse</ansi>.`, corpseColor, corpse.Character.Name))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> buries the <ansi fg="%s">%s corpse</ansi>.`, user.Character.Name, corpseColor, corpse.Character.Name), user.UserId)
			return true, nil

		}

		return true, nil
	}

	user.SendText(fmt.Sprintf("You don't see a %s around for burying.", rest))

	return true, nil
}

func (c *CleanupModule) mobBuryCommand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		return true, nil
	}

	if corpse, corpseFound := room.FindCorpse(rest); corpseFound {

		if room.RemoveCorpse(corpse) {

			corpseColor := `mob-corpse`
			if corpse.UserId > 0 {
				corpseColor = `user-corpse`
			}

			room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> buries the <ansi fg="%s">%s corpse</ansi>.`, mob.Character.Name, corpseColor, corpse.Character.Name))
			return true, nil

		}

		return true, nil
	}

	return true, nil
}
