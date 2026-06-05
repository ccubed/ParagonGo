package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/ansitags"
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	g := GMCPHelpModule{
		plug: plugins.New(`gmcp.Help`, `1.0`),
	}

	events.RegisterListener(GMCPHelpRequest{}, g.helpRequestHandler)
}

type GMCPHelpModule struct {
	plug *plugins.Plugin
}

// GMCPHelpRequest is fired when the web client sends !!GMCP(Help <topic>)
type GMCPHelpRequest struct {
	UserId int
	Topic  string
}

func (g GMCPHelpRequest) Type() string { return `GMCPHelpRequest` }

// GMCPHelpPayload is the GMCP payload sent back to the client.
type GMCPHelpPayload struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Format string `json:"format"` // "terminal" or "html"
}

func (g *GMCPHelpModule) helpRequestHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPHelpRequest)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPHelpRequest", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Continue
	}

	title := evt.Topic
	body, err := usercommands.GetHelpContents(evt.Topic)
	if err != nil || body == `` {
		body = `No help found for "` + evt.Topic + `".`
	}

	if title == `` {
		title = `Help`
	}

	events.AddToQueue(GMCPOut{
		UserId: evt.UserId,
		Module: `Help`,
		Payload: GMCPHelpPayload{
			Title:  title,
			Body:   ansitags.Parse(body),
			Format: `terminal`,
		},
	})

	return events.Continue
}
