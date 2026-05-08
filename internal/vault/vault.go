package vault

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var (
	// ErrVaultLocked reports that the vault is locked and requires authentication.
	ErrVaultLocked = errors.New("vault is locked")

	// ErrVaultNotInitialized reports that the vault has not been initialized.
	ErrVaultNotInitialized = errors.New("vault not initialized")

	// ErrSecretNotFound reports that the requested secret does not exist.
	ErrSecretNotFound = errors.New("secret not found")
)

// Vault provides secure storage for sensitive data using Argon2id + AES-256-GCM.
// The vault must be unlocked with a password before use.
type Vault struct {
	db     *sql.DB
	crypto *Crypto
	locked bool
}

// Open opens an existing vault database.
// The vault is initially locked and requires Unlock() before use.
func Open(path string) (*Vault, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open vault db: %w", err)
	}

	// Configure connection
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Enable full synchronous mode for vault
	if _, err := db.Exec("PRAGMA synchronous=FULL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set synchronous: %w", err)
	}

	return &Vault{
		db:     db,
		locked: true,
	}, nil
}

// Initialize creates a new vault with the given password.
// This should only be called once during initial setup.
func Initialize(path, password string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open vault db: %w", err)
	}
	defer db.Close()

	// Create schema
	schema := `
		CREATE TABLE IF NOT EXISTS vault_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS secrets (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			created_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Generate salt
	salt, err := GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	// Store salt
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	if _, err := db.Exec("INSERT INTO vault_meta (key, value) VALUES (?, ?)", "salt", saltB64); err != nil {
		return fmt.Errorf("store salt: %w", err)
	}

	// Store password hash for verification
	passwordHash := HashPassword(password, salt)
	if _, err := db.Exec("INSERT INTO vault_meta (key, value) VALUES (?, ?)", "password_hash", passwordHash); err != nil {
		return fmt.Errorf("store password hash: %w", err)
	}

	return nil
}

// Unlock unlocks the vault with the provided password.
// Returns ErrInvalidPassword if the password is incorrect.
func (v *Vault) Unlock(password string) error {
	// Load salt
	var saltB64 string
	err := v.db.QueryRow("SELECT value FROM vault_meta WHERE key = ?", "salt").Scan(&saltB64)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrVaultNotInitialized
		}
		return fmt.Errorf("load salt: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}

	// Verify password
	var expectedHash string
	err = v.db.QueryRow("SELECT value FROM vault_meta WHERE key = ?", "password_hash").Scan(&expectedHash)
	if err != nil {
		return fmt.Errorf("load password hash: %w", err)
	}

	if !VerifyPassword(password, salt, expectedHash) {
		return ErrInvalidPassword
	}

	// Initialize crypto
	crypto, err := NewCrypto(password, salt)
	if err != nil {
		return fmt.Errorf("initialize crypto: %w", err)
	}

	v.crypto = crypto
	v.locked = false

	return nil
}

// Lock locks the vault and clears the master key from memory.
func (v *Vault) Lock() {
	if v.crypto != nil {
		// Zero out master key
		for i := range v.crypto.masterKey {
			v.crypto.masterKey[i] = 0
		}
		v.crypto = nil
	}
	v.locked = true
}

// IsLocked returns true if the vault is locked.
func (v *Vault) IsLocked() bool {
	return v.locked
}

// Set stores an encrypted secret in the vault.
func (v *Vault) Set(ctx context.Context, key string, value []byte) error {
	if v.locked {
		return ErrVaultLocked
	}

	// Encrypt value
	encrypted, err := v.crypto.Encrypt(value)
	if err != nil {
		return fmt.Errorf("encrypt value: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Upsert secret
	query := `
		INSERT INTO secrets (key, value, created_at_utc, updated_at_utc)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at_utc = excluded.updated_at_utc
	`

	_, err = v.db.ExecContext(ctx, query, key, encrypted, now, now)
	if err != nil {
		return fmt.Errorf("store secret: %w", err)
	}

	return nil
}

// Get retrieves and decrypts a secret from the vault.
func (v *Vault) Get(ctx context.Context, key string) ([]byte, error) {
	if v.locked {
		return nil, ErrVaultLocked
	}

	var encrypted string
	err := v.db.QueryRowContext(ctx, "SELECT value FROM secrets WHERE key = ?", key).Scan(&encrypted)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("load secret: %w", err)
	}

	// Decrypt value
	plaintext, err := v.crypto.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt value: %w", err)
	}

	return plaintext, nil
}

// Delete removes a secret from the vault.
func (v *Vault) Delete(ctx context.Context, key string) error {
	if v.locked {
		return ErrVaultLocked
	}

	result, err := v.db.ExecContext(ctx, "DELETE FROM secrets WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return ErrSecretNotFound
	}

	return nil
}

// List returns all secret keys in the vault.
func (v *Vault) List(ctx context.Context) ([]string, error) {
	if v.locked {
		return nil, ErrVaultLocked
	}

	rows, err := v.db.QueryContext(ctx, "SELECT key FROM secrets ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate keys: %w", err)
	}

	return keys, nil
}

// Close closes the vault database.
func (v *Vault) Close() error {
	v.Lock()
	return v.db.Close()
}

// ChangePassword changes the vault password.
// This re-encrypts all secrets with a new master key.
func (v *Vault) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	if v.locked {
		return ErrVaultLocked
	}

	// Verify old password
	var saltB64 string
	err := v.db.QueryRow("SELECT value FROM vault_meta WHERE key = ?", "salt").Scan(&saltB64)
	if err != nil {
		return fmt.Errorf("load salt: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}

	var expectedHash string
	err = v.db.QueryRow("SELECT value FROM vault_meta WHERE key = ?", "password_hash").Scan(&expectedHash)
	if err != nil {
		return fmt.Errorf("load password hash: %w", err)
	}

	if !VerifyPassword(oldPassword, salt, expectedHash) {
		return ErrInvalidPassword
	}

	// Generate new salt
	newSalt, err := GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate new salt: %w", err)
	}

	// Create new crypto instance
	newCrypto, err := NewCrypto(newPassword, newSalt)
	if err != nil {
		return fmt.Errorf("create new crypto: %w", err)
	}

	// Read and decrypt all secrets before opening the write transaction. The
	// vault uses a single SQLite connection; calling Get while a transaction is
	// open can deadlock waiting for another connection.
	keys, err := v.List(ctx)
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}
	plaintextSecrets := make(map[string][]byte, len(keys))
	for _, key := range keys {
		plaintext, err := v.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("decrypt secret %s: %w", key, err)
		}
		plaintextSecrets[key] = plaintext
	}

	tx, err := v.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, key := range keys {
		// Encrypt with new key
		encrypted, err := newCrypto.Encrypt(plaintextSecrets[key])
		if err != nil {
			return fmt.Errorf("encrypt secret %s: %w", key, err)
		}

		// Update secret
		_, err = tx.ExecContext(ctx, "UPDATE secrets SET value = ?, updated_at_utc = ? WHERE key = ?",
			encrypted, time.Now().UTC().Format(time.RFC3339Nano), key)
		if err != nil {
			return fmt.Errorf("update secret %s: %w", key, err)
		}
	}

	// Update salt and password hash
	newSaltB64 := base64.StdEncoding.EncodeToString(newSalt)
	newPasswordHash := HashPassword(newPassword, newSalt)

	_, err = tx.Exec("UPDATE vault_meta SET value = ? WHERE key = ?", newSaltB64, "salt")
	if err != nil {
		return fmt.Errorf("update salt: %w", err)
	}

	_, err = tx.Exec("UPDATE vault_meta SET value = ? WHERE key = ?", newPasswordHash, "password_hash")
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	// Update crypto instance
	v.crypto = newCrypto

	return nil
}
