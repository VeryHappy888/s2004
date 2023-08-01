package store

import (
	"ws-go/libsignal/state/record"
)

// SignedPreKey store is an iface that describes how to persistently
// store signed PreKeys.
type SignedPreKey interface {
	// LoadSignedPreKey loads a local SignedPreKeyRecord
	LoadSignedPreKey(signedPreKeyID uint32) *record.SignedPreKey

	// LoadSignedPreKeys loads all local SignedPreKeyRecords
	LoadSignedPreKeys() []*record.SignedPreKey

	// Store a local SignedPreKeyRecord
	StoreSignedPreKey(signedPreKeyID uint32, record *record.SignedPreKey)

	// CheckNil to see if store contains the given record
	ContainsSignedPreKey(signedPreKeyID uint32) bool

	// Delete a SignedPreKeyRecord from local storage
	RemoveSignedPreKey(signedPreKeyID uint32)
}
