package gmcp

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	g := GMCPGameModule{
		plug: plugins.New(`gmcp.Game`, `1.0`),
	}

	events.RegisterListener(events.PlayerDespawn{}, g.onJoinLeave)
	events.RegisterListener(events.PlayerSpawn{}, g.onJoinLeave)
	events.RegisterListener(events.PlayerChanged{}, g.onJoinLeave)
	events.RegisterListener(events.LevelUp{}, g.onJoinLeave)
	events.RegisterListener(events.CharacterChanged{}, g.onJoinLeave)
	events.RegisterListener(GMCPGameRequest{}, g.onGameRequest)

}

type GMCPGameModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin
}

type GMCPGameRequest struct {
	UserId int
}

func (g GMCPGameRequest) Type() string { return `GMCPGameRequest` }

func (g *GMCPGameModule) onGameRequest(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPGameRequest)
	if !typeOk {
		return events.Cancel
	}

	g.sendGamePayload(evt.UserId)
	return events.Continue
}

type GMCPGamePlayer struct {
	Username   string `json:"username"`
	Name       string `json:"name"`
	Level      int    `json:"level"`
	Alignment  string `json:"alignment"`
	Profession string `json:"profession"`
	TimeOnline string `json:"time_online"`
	Role       string `json:"role"`
}

func (g *GMCPGameModule) buildWhoPayload() []GMCPGamePlayer {
	players := []GMCPGamePlayer{}
	for _, user := range users.GetAllActiveUsers() {
		info := user.GetOnlineInfo()
		players = append(players, GMCPGamePlayer{
			Username:   info.Username,
			Name:       info.CharacterName,
			Level:      info.Level,
			Alignment:  info.Alignment,
			Profession: info.Profession,
			TimeOnline: info.OnlineTimeStr,
			Role:       info.Role,
		})
	}
	return players
}

func (g *GMCPGameModule) sendGamePayload(targetUserId int) {

	c := configs.GetConfig()
	tFormat := string(c.TextFormats.Time)

	user := users.GetByUserId(targetUserId)
	if user == nil {
		return
	}

	infoStr := `"Info": { "logintime": "` + user.GetConnectTime().Format(tFormat) + `", "name": "` + string(c.Server.MudName) + `" }`

	players := g.buildWhoPayload()
	playersJSON, _ := json.Marshal(players)

	events.AddToQueue(GMCPOut{
		UserId:  targetUserId,
		Module:  `Game`,
		Payload: `{ ` + infoStr + `, "Who": { "Players": ` + string(playersJSON) + ` } }`,
	})
}

func (g *GMCPGameModule) onJoinLeave(e events.Event) events.ListenerReturn {

	c := configs.GetConfig()
	tFormat := string(c.TextFormats.Time)

	players := g.buildWhoPayload()
	playersJSON, _ := json.Marshal(players)
	whoFragment := `"Who": { "Players": ` + string(playersJSON) + ` }`

	for _, user := range users.GetAllActiveUsers() {
		infoStr := `"Info": { "logintime": "` + user.GetConnectTime().Format(tFormat) + `", "name": "` + string(c.Server.MudName) + `" }`
		events.AddToQueue(GMCPOut{
			UserId:  user.UserId,
			Module:  `Game`,
			Payload: `{ ` + infoStr + `, ` + whoFragment + ` }`,
		})
	}

	return events.Continue
}
