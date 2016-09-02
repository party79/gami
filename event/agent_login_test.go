package event

import (
	"testing"

	"github.com/party79/gami"
)

func TestAgentLogin(t *testing.T) {
	fixture := map[string]string{
		"Agent":    "Agent",
		"Uniqueid": "UniqueID",
		"Channel":  "Channel",
	}

	ev := gami.AMIEvent{
		ID:        "AgentLogin",
		Privilege: []string{"all"},
		Params:    fixture,
	}

	evtype := New(&ev)
	if _, ok := evtype.(*AgentLogin); !ok {
		t.Fatal("AgentLogin type assertion")
	}

	testEvent(t, fixture, evtype)
}
