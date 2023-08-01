// Package cipher is a package for common encrypt/decrypt of symmetric key messages.
package cipher

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Decrypt will use the given key, iv, and ciphertext and return
// the plaintext bytes.
func Decrypt(iv, key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(ciphertext, ciphertext)

	unpaddedText, err := pkcs7Unpad(ciphertext, aes.BlockSize)
	if err != nil {
		return nil, err
	}

	return unpaddedText, nil
}

// Encrypt will use the given iv, key, and plaintext bytes
// and return ciphertext bytes.
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

/*
Encrypt is a function that encrypts plaintext with a given key and an optional initialization vector(iv).
*/
func EncryptS(key, iv, plaintext []byte) ([]byte, error) {
	plaintext = pad(plaintext, aes.BlockSize)

	if len(plaintext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("plaintext is not a multiple of the block size: %d / %d", len(plaintext), aes.BlockSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	var ciphertext []byte
	if iv == nil {
		ciphertext = make([]byte, aes.BlockSize+len(plaintext))
		iv := ciphertext[:aes.BlockSize]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return nil, err
		}

		cbc := cipher.NewCBCEncrypter(block, iv)
		cbc.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)
	} else {
		ciphertext = make([]byte, len(plaintext))

		cbc := cipher.NewCBCEncrypter(block, iv)
		cbc.CryptBlocks(ciphertext, plaintext)
	}

	return ciphertext, nil
}

func pad(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// PKCS7 padding.

// PKCS7 errors.
var (
	// ErrInvalidBlockSize indicates hash blocksize <= 0.
	ErrInvalidBlockSize = errors.New("invalid blocksize")

	// ErrInvalidPKCS7Data indicates bad input to PKCS7 pad or unpad.
	ErrInvalidPKCS7Data = errors.New("invalid PKCS7 data (empty or not padded)")

	// ErrInvalidPKCS7Padding indicates PKCS7 unpad fails to bad input.
	ErrInvalidPKCS7Padding = errors.New("invalid padding on input")
)

// pkcs7Pad right-pads the given byte slice with 1 to n bytes, where
// n is the block size. The size of the result is x times n, where x
// is at least 1.
func pkcs7Pad(b []byte, blocksize int) ([]byte, error) {
	if blocksize <= 0 {
		return nil, ErrInvalidBlockSize
	}
	if b == nil || len(b) == 0 {
		return nil, ErrInvalidPKCS7Data
	}
	n := blocksize - (len(b) % blocksize)
	pb := make([]byte, len(b)+n)
	copy(pb, b)
	copy(pb[len(b):], bytes.Repeat([]byte{byte(n)}, n))
	return pb, nil
}

// pkcs7Unpad validates and unpads data from the given bytes slice.
// The returned value will be 1 to n bytes smaller depending on the
// amount of padding, where n is the block size.
func pkcs7Unpad(b []byte, blocksize int) ([]byte, error) {
	if blocksize <= 0 {
		return nil, ErrInvalidBlockSize
	}
	if b == nil || len(b) == 0 {
		return nil, ErrInvalidPKCS7Data
	}
	if len(b)%blocksize != 0 {
		return nil, ErrInvalidPKCS7Padding
	}
	c := b[len(b)-1]
	n := int(c)
	if n == 0 || n > len(b) {
		return nil, ErrInvalidPKCS7Padding
	}
	for i := 0; i < n; i++ {
		if b[len(b)-n+i] != c {
			return nil, ErrInvalidPKCS7Padding
		}
	}
	return b[:len(b)-n], nil
}
