package test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/hkdf"
	"io"
	"testing"
	"ws-go/libsignal/kdf"
	signalProto "ws-go/libsignal/protos"
)

func pkcs7Pad(b []byte, blocksize int) ([]byte, error) {
	if blocksize <= 0 {
		return nil, errors.New("")
	}
	if b == nil || len(b) == 0 {
		return nil, errors.New("")
	}
	n := blocksize - (len(b) % blocksize)
	pb := make([]byte, len(b)+n)
	copy(pb, b)
	copy(pb[len(b):], bytes.Repeat([]byte{byte(n)}, n))
	return pb, nil
}

func Encrypt(iv, key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	paddedText, err := pkcs7Pad(plaintext, block.BlockSize())
	if err != nil {
		return nil, err
	}
	//paddedText := PKCS5Padding(plaintext,block.BlockSize())
	ciphertext := make([]byte, len(paddedText))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedText)

	return ciphertext, nil
}

/**
PKCS5包装
*/
func PKCS5Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func newKeyMaterial(keyMaterialBytes []byte) *kdf.KeyMaterial {
	cipherKey := keyMaterialBytes[:32] // Use the first 32 bytes of the key material for the CipherKey
	macKey := keyMaterialBytes[32:64]  // Use bytes 32-64 of the key material for the MacKey
	iv := keyMaterialBytes[64:80]      // Use the last 16 bytes for the IV.

	keyMaterial := kdf.KeyMaterial{
		CipherKey: cipherKey,
		MacKey:    macKey,
		IV:        iv,
	}

	return &keyMaterial
}

// DeriveSecrets derives the requested number of bytes using HKDF, given
// the inputKeyMaterial, salt and the info
func DeriveSecrets(inputKeyMaterial, salt, info []byte, size int) ([]byte, error) {
	hkdf := hkdf.New(sha256.New, inputKeyMaterial, salt, info)

	secrets := make([]byte, size)
	n, err := io.ReadFull(hkdf, secrets)
	if err != nil {
		return nil, err
	}
	if n != size {
		return nil, err
	}
	return secrets, nil
}

func TestEncrypted(t *testing.T) {
	keyMaterialBytes, err := hex.DecodeString("a08fc8df701293246b1ac010d1c3cb051244ec8009c60347737a5642c0c0a30f473b34a412e69f8c803d89b5d4ff0311d584849153208056f02386652f090821f04e734759317e0ebdc9cb5513aa880c")
	if err != nil {
		t.Fatal(err)
	}
	enccryptData := "0a01500b0b0b0b0b0b0b0b0b0b0b"
	enccryptBytes, _ := hex.DecodeString(enccryptData)
	material := newKeyMaterial(keyMaterialBytes)
	d, _ := Encrypt(material.IV, material.CipherKey, enccryptBytes)
	t.Log(hex.EncodeToString(d))
}

func TestDeriveSecrets(t *testing.T) {
	inputKeyMaterial := "19d7cc42618bed538d1017b52a25a96d33f52a917931b83b765c26ad67faff6d"
	info := "5768697370657252617463686574"
	salt := "c4f69a297292eb320af2f377a39b7b1c1c80670e707af57b58c18d6c6a07f3fd"
	inputData, _ := hex.DecodeString(inputKeyMaterial)
	infoData, _ := hex.DecodeString(info)
	saltData, _ := hex.DecodeString(salt)
	t.Log(string(infoData))
	d, _ := DeriveSecrets(inputData, saltData, infoData, 64)
	t.Log(hex.EncodeToString(d))
}

func TestPreKeySignalMessage(t *testing.T) {
	//preKeySignaMessage, err := base64.StdEncoding.DecodeString("MwohBUBKTCARsqn1++WRW3iugTXQrfpffWdtR+OF7k4QXLZ6EAAYACIQ4F9LY6xcz02p4PAjo/2o0yFevawuAWR2")
	//t.Log(hex.EncodeToString(preKeySignaMessage))
	preKeySignaMessage, err := hex.DecodeString("0ad30208031221052367ac4923657187e382906d0b203449b650e264f2a08002aa9f514492adae591a2105282c60808cc705d43726c366e3a76c3f4d2ecad5012588436e87d27cdc954e482220c84f43640fefa48fd22950a94faf805f892f2e3ea64cf3f8a5f4e05e59863db62800326b0a2105941a7e5491a8d3f38b616d7571c4548d836852af89fe5e16882921b72988367d122030f1b948e39aa7953bda11ebadd531366074ec6b63d2de6fe9bcedf397bf9f491a2408001220c58a611fb8a22d7da809949bb3dead918f7d3156744a744e1b42b074d713e9b63a490a2105c1116de188d23e15ea7b275ff412c8e6bcbf764bff90cc1610ea00748299be651a240801122036e006a5b9c941754e5f61ad53f3b561614043413e131c6a8cfa241d033055e950c1a1c68202589fb1b294066a2105ee9087ba4b18c672b85d777e8af2896ba99037f401597a70003056cef7fbc24a12fd0208031221052367ac4923657187e382906d0b203449b650e264f2a08002aa9f514492adae591a2105282c60808cc705d43726c366e3a76c3f4d2ecad5012588436e87d27cdc954e482220ba1447a107edbdc0818d8cdaec6474001ba763caed2844baac0201536c314113326b0a2105494be91967187f997ffcaf7082db6102feae0e4c2d60c5018165a3b9e5f85f55122038d916c4163bbc310e43cbc01df01cc86e44a55c85eafe97c4d41a0d70ce69621a2408031220b81cbcbd660d378a4e7cd046b70e73cf63f20c30cf2c84aeca77cf3bb57ec3ca3a490a2105e565340dc3f0b3d8fdc677adfc45fcce4583ee79ad4d58d4ff1f1ce702ed22721a2408001220b1d58b7bcf618441d3f57a2f4fabb162e790dceb2acb5a4911c516603c2dadab4a2a08371221055876325106f44c8f12ca7c73b0eba5cfc2b2144a97a189c5368663497942475618d68ea20350c1a1c68202589fb1b294066a21055876325106f44c8f12ca7c73b0eba5cfc2b2144a97a189c536866349794247561200")
	if err != nil {
		t.Fatal(err)
	}
	protoPreKeySignalMessage := &signalProto.SessionStructure{}

	err = proto.Unmarshal(preKeySignaMessage, protoPreKeySignalMessage)
	if err != nil {
		t.Fatal(err)
	}

	d, err := json.MarshalIndent(protoPreKeySignalMessage, " ", " ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(d))
}
