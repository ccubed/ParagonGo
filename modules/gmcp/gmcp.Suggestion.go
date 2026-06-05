package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/suggestions"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	g := GMCPSuggestionModule{
		plug: plugins.New(`gmcp.Suggestion`, `1.0`),
	}
	events.RegisterListener(GMCPSuggestionRequest{}, g.suggestionRequestHandler)
}

type GMCPSuggestionModule struct {
	plug *plugins.Plugin
}

// GMCPSuggestionRequest is fired when the web client sends !!GMCP(Suggestion <partial-text>)
type GMCPSuggestionRequest struct {
	UserId      int
	PartialText string
}

func (g GMCPSuggestionRequest) Type() string { return `GMCPSuggestionRequest` }

// GMCPSuggestionPayload is the GMCP payload sent back to the client.
// Input is the original partial text the user typed.
// Suggestions is the list of fully-completed strings the client may cycle through.
type GMCPSuggestionPayload struct {
	Input       string   `json:"input"`
	Suggestions []string `json:"suggestions"`
}

func (g *GMCPSuggestionModule) suggestionRequestHandler(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(GMCPSuggestionRequest)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "GMCPSuggestionRequest", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}

	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Continue
	}

	// GetAutoComplete returns suffixes (the characters to append to the partial
	// input). Build the full completed strings for the client.
	suffixes := suggestions.GetAutoComplete(evt.UserId, evt.PartialText)
	completed := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		completed = append(completed, evt.PartialText+suffix)
	}

	events.AddToQueue(GMCPOut{
		UserId: evt.UserId,
		Module: `Suggestion`,
		Payload: GMCPSuggestionPayload{
			Input:       evt.PartialText,
			Suggestions: completed,
		},
	})

	return events.Continue
}
