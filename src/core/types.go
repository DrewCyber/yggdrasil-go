package core

// In-band packet types
const (
	typeSessionDummy = iota // nolint:deadcode,varcheck
	typeSessionTraffic
	typeSessionProto
)

// Protocol packet types
const (
	typeProtoDummy = iota
	typeProtoNodeInfoRequest
	typeProtoNodeInfoResponse
	typeProtoDebug = 255
)

// PeerChangeCallback is called when the peer connection state changes.
// This callback fires whenever a peer connects, disconnects, or is added/removed.
// The connected parameter indicates the number of currently connected (Up) peers.
// The total parameter indicates the total number of configured peers.
type PeerChangeCallback func(connected int, total int)
