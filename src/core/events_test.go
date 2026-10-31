package core

import (
	"net/url"
	"testing"
	"time"

	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

// waitForEvent returns the next event matching up/inbound, skipping
// non-matching ones (e.g. the add() entry-created event).
func waitForEvent(t *testing.T, ch chan PeerEvent, up, inbound bool) PeerEvent {
	t.Helper()
	for {
		select {
		case ev := <-ch:
			if ev.Up == up && ev.Inbound == inbound {
				return ev
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for event Up=%v Inbound=%v", up, inbound)
		}
	}
}

// TestSetPeerListener checks that peer connect/disconnect transitions
// are delivered to the listener registered via SetPeerListener: an
// outbound up on the dialing node, an inbound up on the listening node,
// and a down on each side when one node stops.
func TestSetPeerListener(t *testing.T) {
	cfgA, cfgB := config.GenerateConfig(), config.GenerateConfig()
	if err := cfgA.GenerateSelfSignedCertificate(); err != nil {
		t.Fatal(err)
	}
	if err := cfgB.GenerateSelfSignedCertificate(); err != nil {
		t.Fatal(err)
	}
	logger := GetLoggerWithPrefix("", false)

	nodeA, err := New(cfgA.Certificate, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer nodeA.Stop()
	nodeB, err := New(cfgB.Certificate, logger)
	if err != nil {
		t.Fatal(err)
	}

	eventsA := make(chan PeerEvent, 16)
	eventsB := make(chan PeerEvent, 16)
	nodeA.SetPeerListener(func(ev PeerEvent) { eventsA <- ev })
	nodeB.SetPeerListener(func(ev PeerEvent) { eventsB <- ev })

	nodeAListenURL, err := url.Parse("tcp://localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	nodeAListener, err := nodeA.Listen(nodeAListenURL, "")
	if err != nil {
		t.Fatal(err)
	}
	nodeAURL, err := url.Parse("tcp://" + nodeAListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err = nodeB.CallPeer(nodeAURL, ""); err != nil {
		t.Fatal(err)
	}

	// nodeB dials out; nodeA sees the connection come in.
	waitForEvent(t, eventsB, true, false)
	waitForEvent(t, eventsA, true, true)

	// Stopping nodeB tears the link down on both sides.
	nodeB.Stop()
	waitForEvent(t, eventsB, false, false)
	waitForEvent(t, eventsA, false, true)
}
