package store

import (
	"ws-go/libsignal/groups/state/store"
)

// SignalProtocol store is an iface that implements the
// methods for all stores needed in the Signal Protocol.
type SignalProtocol interface {
	IdentityKey
	PreKey
	Session
	SignedPreKey
	store.SenderKey
}
