package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/scripting"
)

// UserScriptLogin fires the global user script's onLogin handler when a player
// enters the world.
func UserScriptLogin(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Continue
	}

	scripting.TryUserScriptEvent(`onLogin`, evt.UserId)

	return events.Continue
}

// UserScriptLogout fires the global user script's onLogout handler when a
// player leaves the world. It must run before the final HandleLeave listener
// so the user record is still resolvable.
func UserScriptLogout(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		return events.Continue
	}

	scripting.TryUserScriptEvent(`onLogout`, evt.UserId)

	return events.Continue
}

// UserScriptLevelUp fires the global user script's onLevel handler when a
// player gains a level.
func UserScriptLevelUp(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.LevelUp)
	if !typeOk {
		return events.Continue
	}

	scripting.TryUserLevelEvent(evt.UserId, map[string]any{
		`newLevel`:       evt.NewLevel,
		`previousLevel`:  evt.NewLevel - evt.LevelsGained,
		`levelsGained`:   evt.LevelsGained,
		`trainingPoints`: evt.TrainingPoints,
		`statPoints`:     evt.StatPoints,
		`livesGained`:    evt.LivesGained,
	})

	return events.Continue
}
