package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/argon2"
	_ "modernc.org/sqlite"
)

var errInvalidPassword = errors.New("invalid vault password")

type migration struct {
	version int
	hash    string
	query   string
}

var vaultMigrations = []migration{{
	version: 1,
	query: `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	hash TEXT NOT NULL,
	applied_at_utc TEXT NOT NULL,
	agent_version TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS vault_metadata (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	salt BLOB NOT NULL,
	verifier_nonce BLOB NOT NULL,
	verifier_ciphertext BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS trusted_plugin_keys (
	fingerprint TEXT PRIMARY KEY,
	public_key BLOB NOT NULL,
	added_at_utc TEXT NOT NULL
);
`,
	hash: "vault-v1-20260429",
}}

// Store owns the encrypted vault metadata and trusted plugin keys.
type Store struct {
	db  *sql.DB
	key []byte
}

// OpenOrCreate opens or initializes the vault.
func OpenOrCreate(path string, password string, agentVersion string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if fileExists(path) {
		needsMigration, err := hasPendingMigrations(path)
		if err != nil {
			return nil, err
		}
		if needsMigration {
			if err := createVaultBackup(filepath.Dir(path), path); err != nil {
				return nil, err
			}
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := applyMigrations(db, agentVersion); err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.bootstrapOrUnlock(password); err != nil {
		return nil, err
	}
	return store, nil
}

// Close closes the vault.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) bootstrapOrUnlock(password string) error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM vault_metadata`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return err
		}
		key := deriveKey(password, salt)
		nonce, ciphertext, err := seal(key, []byte("kairos-vault-verifier"))
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO vault_metadata (id, salt, verifier_nonce, verifier_ciphertext) VALUES (1, ?, ?, ?)`, salt, nonce, ciphertext); err != nil {
			return err
		}
		s.key = key
		return nil
	}
	var salt, nonce, ciphertext []byte
	if err := s.db.QueryRow(`SELECT salt, verifier_nonce, verifier_ciphertext FROM vault_metadata WHERE id = 1`).Scan(&salt, &nonce, &ciphertext); err != nil {
		return err
	}
	key := deriveKey(password, salt)
	plaintext, err := openSealed(key, nonce, ciphertext)
	if err != nil {
		return errInvalidPassword
	}
	if string(plaintext) != "kairos-vault-verifier" {
		return errInvalidPassword
	}
	s.key = key
	return nil
}

// ValidatePassword verifies the provided vault password.
func (s *Store) ValidatePassword(password string) error {
	var salt, nonce, ciphertext []byte
	if err := s.db.QueryRow(`SELECT salt, verifier_nonce, verifier_ciphertext FROM vault_metadata WHERE id = 1`).Scan(&salt, &nonce, &ciphertext); err != nil {
		return err
	}
	key := deriveKey(password, salt)
	plaintext, err := openSealed(key, nonce, ciphertext)
	if err != nil {
		return errInvalidPassword
	}
	if string(plaintext) != "kairos-vault-verifier" {
		return errInvalidPassword
	}
	return nil
}

// StoreTrustedPluginKey persists a trusted plugin public key.
func (s *Store) StoreTrustedPluginKey(ctx context.Context, key TrustedPluginKey) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO trusted_plugin_keys (fingerprint, public_key, added_at_utc)
VALUES (?, ?, ?)
ON CONFLICT(fingerprint) DO UPDATE SET public_key = excluded.public_key, added_at_utc = excluded.added_at_utc`,
		key.Fingerprint,
		key.PublicKey,
		key.AddedAtUTC.UTC().Format(time.RFC3339Nano),
	)
	return err
}

// TrustedPluginKey returns a trusted plugin key if present.
func (s *Store) TrustedPluginKey(ctx context.Context, fingerprint string) (*TrustedPluginKey, error) {
	var out TrustedPluginKey
	var addedAt string
	err := s.db.QueryRowContext(ctx, `SELECT fingerprint, public_key, added_at_utc FROM trusted_plugin_keys WHERE fingerprint = ?`, fingerprint).Scan(&out.Fingerprint, &out.PublicKey, &addedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out.AddedAtUTC, err = time.Parse(time.RFC3339Nano, addedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
}

func seal(key []byte, plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, plaintext, nil), nil
}

func openSealed(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func applyMigrations(db *sql.DB, agentVersion string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, hash TEXT NOT NULL, applied_at_utc TEXT NOT NULL, agent_version TEXT NOT NULL);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	var currentVersion int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&currentVersion); err != nil {
		return err
	}
	for _, migration := range vaultMigrations {
		if migration.version <= currentVersion {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migration.query); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, hash, applied_at_utc, agent_version) VALUES (?, ?, ?, ?)`, migration.version, migration.hash, time.Now().UTC().Format(time.RFC3339Nano), agentVersion); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func hasPendingMigrations(path string) (bool, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return false, err
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, hash TEXT NOT NULL, applied_at_utc TEXT NOT NULL, agent_version TEXT NOT NULL);`); err != nil {
		return false, err
	}
	var currentVersion int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&currentVersion); err != nil {
		return false, err
	}
	latest := 0
	for _, migration := range vaultMigrations {
		if migration.version > latest {
			latest = migration.version
		}
	}
	return currentVersion < latest, nil
}

func createVaultBackup(stateDir string, source string) error {
	backupDir := filepath.Join(stateDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(backupDir, fmt.Sprintf("pre-migrate-%s-%d.bak", filepath.Base(source), time.Now().UTC().Unix()))
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
