package store

import (
	"ws-go/libsignal/state/record"
)

// PreKey store is an iface describing the local storage
// of PreKeyRecords
type PreKey interface {
	// Load a local PreKeyRecord
	LoadPreKey(preKeyID uint32) *record.PreKey

	// Store a local PreKeyRecord
	StorePreKey(preKeyID uint32, preKeyRecord *record.PreKey)

	StorePreKeyIds(preKeys []*record.PreKey)

	// CheckNil to see if the store contains a PreKeyRecord
	ContainsPreKey(preKeyID uint32) bool

	// Delete a PreKeyRecord from local storage.
	RemovePreKey(preKeyID uint32)
}
