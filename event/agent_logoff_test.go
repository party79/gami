package event

import (
	"testing"

	"github.com/party79/gami"
)

func TestAgentLogoff(t *testing.T) {
	fixture := map[string]string{
		"Agent":     "Agent",
		"Uniqueid":  "UniqueID",
		"Logintime": "LoginTime",
	}

	ev := gami.AMIEvent{
		ID:        "AgentLogoff",
		Privilege: []string{"all"},
		Params:    fixture,
	}

	evtype := New(&ev)
	if _, ok := evtype.(*AgentLogoff); !ok {
		t.Fatal("AgentLogoff type assertion")
	}

	testEvent(t, fixture, evtype)
}
