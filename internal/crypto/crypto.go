package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// WrapKey encrypts a File Encryption Key (FEK) using a Server Wrapping Key (SWK)
// It uses AES-256-GCM. The SWK is hashed with SHA-256 to ensure it is 32 bytes.
func WrapKey(fek []byte, swk string) (string, error) {
	key := sha256.Sum256([]byte(swk))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Ciphertext is nonce + sealed fek
	sealed := gcm.Seal(nonce, nonce, fek, nil)
	return fmt.Sprintf("%x", sealed), nil
}

// UnwrapKey decrypts a wrapped FEK using the SWK
func UnwrapKey(wrappedHex string, swk string) ([]byte, error) {
	key := sha256.Sum256([]byte(swk))
	var wrapped []byte
	_, err := fmt.Sscanf(wrappedHex, "%x", &wrapped)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(wrapped) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := wrapped[:nonceSize], wrapped[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap key: %w", err)
	}

	return plaintext, nil
}
