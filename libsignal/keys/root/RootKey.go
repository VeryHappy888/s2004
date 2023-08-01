// Package root provides root keys which are used to derive new chain and
// root keys in a ratcheting session.
package root

import (
	"encoding/hex"
	"log"
	"ws-go/libsignal/ecc"
	"ws-go/libsignal/kdf"
	"ws-go/libsignal/keys/chain"
	"ws-go/libsignal/keys/session"
	"ws-go/libsignal/util/bytehelper"
)

// DerivedSecretsSize is the size of the derived secrets for root keys.
const DerivedSecretsSize = 64

// KdfInfo is used as the info for message keys to derive secrets using a Key Derivation Function
const KdfInfo string = "WhisperRatchet"

// NewKey returns a new RootKey given the key derivation function and bytes.
func NewKey(kdf kdf.HKDF, key []byte) *Key {
	rootKey := Key{
		kdf: kdf,
		key: key,
	}

	return &rootKey
}

// Key is a structure for RootKeys, which are used to derive a new set of chain and root
// keys for every round trip of messages.
type Key struct {
	kdf kdf.HKDF
	key []byte
}

// Bytes returns the RootKey in bytes.
func (k *Key) Bytes() []byte {
	return k.key
}

// CreateChain creates a new RootKey and ChainKey from the recipient's ratchet key and our private key.
func (k *Key) CreateChain(theirRatchetKey ecc.ECPublicKeyable, ourRatchetKey *ecc.ECKeyPair) (*session.KeyPair, error) {
	var keyMaterial []byte
	theirPublicKey := theirRatchetKey.PublicKey()
	ourPrivateKey := ourRatchetKey.PrivateKey().Serialize()

	// Use our key derivation function to calculate a shared secret.
	sharedSecret := kdf.CalculateSharedSecret(theirPublicKey, ourPrivateKey)
	// 这里有问题导致 发送消息失败
	//copy(keyMaterial[:], sharedSecret[:])
	// 解决
	keyMaterial = bytehelper.ArrayToSlice(sharedSecret)
	derivedSecretBytes, err := kdf.DeriveSecrets(keyMaterial, k.key, []byte(KdfInfo), DerivedSecretsSize)
	if err != nil {
		return nil, err
	}
	log.Println("derivedSecretBytes", hex.EncodeToString(derivedSecretBytes))
	// Split the derived secret bytes in half, using one half for the root key and the second for the chain key.
	derivedSecrets := session.NewDerivedSecrets(derivedSecretBytes)

	// Create new root and chain key structures from the derived secrets.
	rootKey := NewKey(k.kdf, derivedSecrets.RootKey())
	chainKey := chain.NewKey(k.kdf, derivedSecrets.ChainKey(), 0)

	// Create a session keypair with the generated root and chain keys.
	keyPair := session.NewKeyPair(
		rootKey,
		chainKey,
	)

	return keyPair, nil
}