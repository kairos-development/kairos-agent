package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

var (
	// ErrInvalidPassword reports that the provided password is incorrect.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrInvalidCiphertext reports that the ciphertext is malformed or corrupted.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

const (
	// Argon2id parameters following OWASP recommendations for 2024+
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32 // 256 bits for AES-256

	saltLen  = 32 // 256 bits
	nonceLen = 12 // GCM standard nonce size
)

// Crypto provides encryption and decryption operations using Argon2id + AES-256-GCM.
// The master key and salts never leave local storage.
type Crypto struct {
	masterKey []byte
}

// NewCrypto creates a new crypto instance with a derived master key.
// The password is hashed using Argon2id with a random salt.
func NewCrypto(password string, salt []byte) (*Crypto, error) {
	if len(salt) != saltLen {
		return nil, fmt.Errorf("salt must be %d bytes", saltLen)
	}

	// Derive master key using Argon2id
	masterKey := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	return &Crypto{
		masterKey: masterKey,
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext with embedded nonce.
func (c *Crypto) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(c.masterKey)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM.
// Returns plaintext or ErrInvalidPassword if authentication fails.
func (c *Crypto) Decrypt(ciphertextB64 string) ([]byte, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	if len(ciphertext) < nonceLen {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(c.masterKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// Extract nonce from ciphertext
	nonce := ciphertext[:nonceLen]
	ciphertext = ciphertext[nonceLen:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	return plaintext, nil
}

// GenerateSalt generates a cryptographically secure random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// HashPassword creates a one-way hash of a password for verification.
// This is used for password change validation, not for encryption.
func HashPassword(password string, salt []byte) string {
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)
	return base64.StdEncoding.EncodeToString(hash)
}

// VerifyPassword verifies a password against a stored hash.
func VerifyPassword(password string, salt []byte, expectedHash string) bool {
	actualHash := HashPassword(password, salt)
	return actualHash == expectedHash
}

// DeriveKeyFingerprint creates a deterministic fingerprint of the master key.
// This is used to verify key integrity without exposing the key.
func (c *Crypto) DeriveKeyFingerprint() string {
	hash := sha256.Sum256(c.masterKey)
	return base64.StdEncoding.EncodeToString(hash[:16]) // First 128 bits
}
