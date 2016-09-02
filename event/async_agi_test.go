package event

import (
	"testing"

	"github.com/bit4bit/gami"
)

func TestAsyncAGI(t *testing.T) {
	fixture := map[string]string{
		"Event":     "Event",
		"Channel":   "Channel",
		"Subevent":  "Subevent",
		"Commandid": "Commandid",
		"Result":    "200%20result%3D0%0A",
		"Env":       "Env%3A%20Env%0A",
	}

	ev := gami.AMIEvent{
		ID:        "AsyncAGI",
		Privilege: []string{"all"},
		Params:    fixture,
	}

	evtype := New(&ev)
	if _, ok := evtype.(*AsyncAGI); !ok {
		t.Log("AsyncAGI type assertion")
		t.Fail()
	}

	testEvent(t, fixture, evtype)
}
