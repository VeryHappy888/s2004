package ecc

import (
	"crypto/aes"
	"crypto/cipher"
)

func AesGcmEncrypt(key, nonce, aad, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Seal(nil, nonce, data, aad), nil
}
