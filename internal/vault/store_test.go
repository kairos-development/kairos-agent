package vault

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestOpenOrCreateAndValidatePassword(t *testing.T) {
	store, err := OpenOrCreate(t.TempDir()+"/vault.db", "secret", "test")
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	defer store.Close()
	if err := store.ValidatePassword("secret"); err != nil {
		t.Fatalf("validate password: %v", err)
	}
	if err := store.ValidatePassword("wrong"); err == nil {
		t.Fatal("expected invalid password error")
	}
	key := TrustedPluginKey{Fingerprint: "abc", PublicKey: []byte("pub"), AddedAtUTC: time.Now().UTC()}
	if err := store.StoreTrustedPluginKey(context.Background(), key); err != nil {
		t.Fatalf("store key: %v", err)
	}
	loaded, err := store.TrustedPluginKey(context.Background(), "abc")
	if err != nil || loaded == nil {
		t.Fatalf("load key: %v", err)
	}
}

func TestOpenOrCreateReopensExistingVault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.db")
	store, err := OpenOrCreate(path, "secret", "test")
	require.NoError(t, err)
	require.NoError(t, store.Close())

	reopened, err := OpenOrCreate(path, "secret", "test")
	require.NoError(t, err)
	defer reopened.Close()
	assert.NoError(t, reopened.ValidatePassword("secret"))

	_, err = OpenOrCreate(path, "wrong", "test")
	assert.ErrorIs(t, err, errInvalidPassword)
}

func TestTrustedPluginKeyNotFoundAndInvalidTimestamp(t *testing.T) {
	store, err := OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "secret", "test")
	require.NoError(t, err)
	defer store.Close()

	missing, err := store.TrustedPluginKey(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, missing)

	_, err = store.db.ExecContext(context.Background(), `INSERT INTO trusted_plugin_keys (fingerprint, public_key, added_at_utc) VALUES (?, ?, ?)`, "bad", []byte("pub"), "not-time")
	require.NoError(t, err)
	_, err = store.TrustedPluginKey(context.Background(), "bad")
	assert.Error(t, err)
}

func TestVaultStorePendingMigrationsAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, hash TEXT NOT NULL, applied_at_utc TEXT NOT NULL, agent_version TEXT NOT NULL)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	pending, err := hasPendingMigrations(path)
	require.NoError(t, err)
	assert.True(t, pending)

	require.NoError(t, createVaultBackup(dir, path))
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
}

func TestSealAndOpenSealedValidation(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	nonce, ciphertext, err := seal(key, []byte("secret"))
	require.NoError(t, err)
	plain, err := openSealed(key, nonce, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, []byte("secret"), plain)

	ciphertext[0] ^= 0xff
	_, err = openSealed(key, nonce, ciphertext)
	assert.Error(t, err)

	_, _, err = seal([]byte("short"), []byte("secret"))
	assert.Error(t, err)
	_, err = openSealed([]byte("short"), nonce, ciphertext)
	assert.Error(t, err)
}
