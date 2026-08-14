package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

type CryptoService interface {
	Decrypt(ciphertextB64 string, nonceB64 string, key []byte) (string, error)
	Encrypt(plaintext string, key []byte) (string, string, error)
	ResolveKey(requestKey string, defaultKey string) []byte
}

type cryptoService struct{}

func NewCryptoService() CryptoService {
	return &cryptoService{}
}

func (s *cryptoService) ResolveKey(requestKey string, defaultKey string) []byte {
	kStr := requestKey
	if kStr == "" {
		kStr = defaultKey
	}
	if kStr == "" {
		return nil
	}

	// If key is 64 hex chars, decode to 32 bytes
	if len(kStr) == 64 {
		if decoded, err := hex.DecodeString(kStr); err == nil && len(decoded) == 32 {
			return decoded
		}
	}

	keyBytes := []byte(kStr)
	// Pad or truncate to 32 bytes for AES-256
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		return padded
	}
	return keyBytes[:32]
}

func (s *cryptoService) Decrypt(ciphertextB64 string, nonceB64 string, key []byte) (string, error) {
	if strings.HasPrefix(ciphertextB64, "enc_") {
		rawB64 := strings.TrimPrefix(ciphertextB64, "enc_")
		if decoded, err := base64.StdEncoding.DecodeString(rawB64); err == nil {
			return string(decoded), nil
		}
	}

	if len(key) == 0 {
		return ciphertextB64, fmt.Errorf("decryption key is empty")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return ciphertextB64, fmt.Errorf("failed to base64 decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return ciphertextB64, fmt.Errorf("failed to base64 decode nonce: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return ciphertextB64, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertextB64, fmt.Errorf("failed to create AES-GCM: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ciphertextB64, fmt.Errorf("failed to decrypt AES-GCM payload: %w", err)
	}

	return string(plaintext), nil
}

func (s *cryptoService) Encrypt(plaintext string, key []byte) (string, string, error) {
	if len(key) == 0 {
		return "", "", fmt.Errorf("encryption key is empty")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create AES-GCM: %w", err)
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)

	return base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(nonce), nil
}
