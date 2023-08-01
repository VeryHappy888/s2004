// Package kdf provides a key derivation function to calculate key output
// and negotiate shared secrets for curve X25519 keys.
package kdf

import (
	"crypto/sha256"
	"encoding/hex"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"io"
	"log"
	"ws-go/libsignal/util/bytehelper"
)

// HKDF is a hashed key derivation function type that can be used to derive keys.
type HKDF func(inputKeyMaterial, salt, info []byte, outputLength int) ([]byte, error)

// DeriveSecrets derives the requested number of bytes using HKDF with the given
// input, salt, and info.
func DeriveSecrets(inputKeyMaterial, salt, info []byte, outputLength int) ([]byte, error) {
	log.Println("inputKeyMaterial:", hex.EncodeToString(inputKeyMaterial))
	log.Println("salt:", hex.EncodeToString(salt))
	log.Println("info:", hex.EncodeToString(info))
	log.Println("outputLength:", outputLength)

	kdf := hkdf.New(sha256.New, inputKeyMaterial, salt, info)

	secrets := make([]byte, outputLength)
	length, err := io.ReadFull(kdf, secrets)
	if err != nil {
		return nil, err
	}
	if length != outputLength {
		return nil, err
	}

	return secrets, nil
}

// CalculateSharedSecret uses DH Curve25519 to find a shared secret. The result of this function
// should be used in `DeriveSecrets` to output the Root and Chain keys.
func CalculateSharedSecret(theirKey, ourKey [32]byte) [32]byte {
	log.Println("CalculateSharedSecret", hex.EncodeToString(bytehelper.ArrayToSlice(theirKey)), hex.EncodeToString(bytehelper.ArrayToSlice(ourKey)))
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &ourKey, &theirKey)
	log.Println("ret", hex.EncodeToString(bytehelper.ArrayToSlice(sharedSecret)))
	return sharedSecret
}

// KeyMaterial is a structure for representing a cipherkey, mac, and iv
type KeyMaterial struct {
	CipherKey []byte
	MacKey    []byte
	IV        []byte
}
