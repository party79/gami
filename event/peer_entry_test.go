package event

import (
	"testing"

	"github.com/party79/gami"
)

func TestPeerEntry(t *testing.T) {
	fixture := map[string]string{
		"Channeltype":    "Channeltype",
		"Objectname":     "Objectname",
		"Chanobjecttype": "Chanobjecttype",
		"Ipaddress":      "Ipaddress",
		"Ipport":         "Ipport",
		"Dynamic":        "Dynamic",
		"Natsupport":     "Natsupport",
		"Videosupport":   "Videosupport",
		"Textsupport":    "Textsupport",
		"Acl":            "Acl",
		"Status":         "Status",
		"Realtimedevice": "Realtimedevice",
	}

	ev := gami.AMIEvent{
		ID:        "PeerEntry",
		Privilege: []string{"all"},
		Params:    fixture,
	}

	evtype := New(&ev)
	if _, ok := evtype.(*PeerEntry); !ok {
		t.Fatal("PeerEntry type assertion")
	}

	testEvent(t, fixture, evtype)
}
