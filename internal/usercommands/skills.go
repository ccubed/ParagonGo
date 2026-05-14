package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type SkillsOptions struct {
	SkillList      map[string]int
	TrainingPoints int
	SkillCooldowns map[string]int
}

func Skills(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	user.SendText(buildSkillsPanel(user))

	tpColor := `red`
	if user.Character.TrainingPoints > 0 {
		tpColor = `yellow`
	}
	user.SendText(fmt.Sprintf(` You have <ansi fg="%s-bold">%d Training Points</ansi>. Level up to earn more.`,
		tpColor, user.Character.TrainingPoints))

	if rest == `extra` {
		user.SendText(`<ansi fg="yellow">Cooldown Tracking:</ansi>`)
		for name, rnds := range user.Character.GetAllCooldowns() {
			user.SendText(fmt.Sprintf(` <ansi fg="yellow">%s</ansi>: <ansi fg="red">%d</ansi>`, name, rnds))
		}
	}

	return true, nil
}
