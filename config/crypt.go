package config

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

// Decrypt decrypts small messages
// golang.org/x/crypto/nacl/secretbox
func Decrypt(message, secret []byte) ([]byte, error) {
	var nonce [24]byte
	var secretKey = sha256.Sum256(secret)
	copy(nonce[:], message[:24])
	decrypted, ok := secretbox.Open(nil, message[24:], &nonce, &secretKey)
	if !ok {
		return nil, fmt.Errorf("decryption error")
	}
	return decrypted, nil
}

// Encrypt encrypts small messages
// golang.org/x/crypto/nacl/secretbox
func Encrypt(message, secret []byte) ([]byte, error) {
	var nonce [24]byte
	var secretKey = sha256.Sum256(secret)
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], message, &nonce, &secretKey), nil
}
