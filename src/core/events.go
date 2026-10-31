package core

import "github.com/Arceliar/phony"

// PeerEvent describes a change in the set of configured peer links: a
// link came up or went down, or a peer entry was added or removed.
type PeerEvent struct {
	URI     string // Peering URI in complete form
	Inbound bool   // True if the connection was initiated by the remote side
	Up      bool   // True if the link is now connected
}

// SetPeerListener registers f to be invoked when the connected-peer set
// changes. Transitions are delivered on their edges only: a link that
// keeps failing to dial does not re-fire while it stays down.
//
// f is called synchronously from the links actor: it must return
// quickly, must not block, and must not call Core methods that wait on
// the links actor (such as GetPeers). The listener is intended to be
// installed once, right after New; installing a listener replaces the
// previous one, and nil disables event delivery.
func (c *Core) SetPeerListener(f func(PeerEvent)) {
	phony.Block(&c.links, func() {
		c.links._peerListener = f
	})
}
