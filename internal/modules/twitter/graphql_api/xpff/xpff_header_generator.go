package xpff

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

type XPFFHeaderGenerator struct {
	baseKey string
}

// deriveKey concatenates baseKey+guestID, hashes it with SHA-256,
// and returns the 32-byte key.
func (x *XPFFHeaderGenerator) deriveKey(guestID string) []byte {
	h := sha256.New()
	h.Write([]byte(x.baseKey + guestID))
	return h.Sum(nil)
}

// GenerateXPFF encrypts plaintext under a derived key for guestID,
// prefixing a random 12-byte nonce, and returns hex(nonce|ciphertext|tag).
func (x *XPFFHeaderGenerator) GenerateXPFF(plaintext, guestID string) (string, error) {
	key := x.deriveKey(guestID)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("rand.Read nonce: %w", err)
	}

	// Seal appends ciphertext||tag to the nonce prefix (we'll prepend nonce ourselves)
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	out := append(nonce, ciphertext...)
	return hex.EncodeToString(out), nil
}

// DecodeXPFF takes hex(nonce|ciphertext|tag), splits it apart,
// and returns the decrypted plaintext (or an error if authentication fails).
func (x *XPFFHeaderGenerator) DecodeXPFF(hexStr, guestID string) (string, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("hex.DecodeString: %w", err)
	}

	key := x.deriveKey(guestID)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ct := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("gcm.Open: %w", err)
	}

	return string(plaintext), nil
}
