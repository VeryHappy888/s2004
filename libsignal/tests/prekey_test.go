package tests

import (
	"testing"
	"ws-go/libsignal/logger"
	"ws-go/libsignal/serialize"
	"ws-go/libsignal/util/keyhelper"
)

// TestPreKeys checks generating prekeys.
func TestPreKeys(t *testing.T) {

	// Create a serializer object that will be used to encode/decode data.
	serializer := serialize.NewSerializer()
	serializer.SignalMessage = &serialize.JSONSignalMessageSerializer{}
	serializer.PreKeySignalMessage = &serialize.JSONPreKeySignalMessageSerializer{}
	serializer.SignedPreKeyRecord = &serialize.JSONSignedPreKeyRecordSerializer{}

	logger.Info("Testing prekey generation...")
	identityKeyPair, err := keyhelper.GenerateIdentityKeyPair()
	if err != nil {
		t.Error(err)
	}

	logger.Info("Generating prekeys")
	preKeys, _ := keyhelper.GeneratePreKeys(0, 100, serializer.PreKeyRecord)
	logger.Info("Generated PreKeys: ", preKeys)

	logger.Info("Generating Signed PreKey")
	signedPreKey, _ := keyhelper.GenerateSignedPreKey(identityKeyPair, 1, serializer.SignedPreKeyRecord)
	logger.Info("Signed PreKey: ", signedPreKey)
}
