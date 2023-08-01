package store

import (
	"ws-go/libsignal/keys/message"
)

// MessageKey store is an iface describing the optional local storage
// of message keys.
type MessageKey interface {
	// Load a local message key by id
	LoadMessageKey(keyID uint32) *message.Keys

	// Store a local message key
	StoreMessageKey(keyID uint32, key *message.Keys)

	// CheckNil to see if the store contains a message key with id.
	ContainsMessageKey(keyID uint32) bool

	// Delete a message key from local storage.
	RemoveMessageKey(keyID uint32)
}
