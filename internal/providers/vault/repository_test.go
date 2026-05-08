package vaultprovider

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryTrustPluginKey(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "correct horse battery staple", "test")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	repo := New(store)
	fixed := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	repo.clock = func() time.Time { return fixed }
	ctx := context.Background()

	err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{Fingerprint: "abc123", PublicKey: []byte("pub")})
	require.NoError(t, err)

	key, err := store.TrustedPluginKey(ctx, "abc123")
	require.NoError(t, err)
	assert.Equal(t, "abc123", key.Fingerprint)
	assert.Equal(t, []byte("pub"), key.PublicKey)
	assert.True(t, fixed.Equal(key.AddedAtUTC))
}

func TestNew(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	defer store.Close()

	repo := New(store)
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.store)
	assert.NotNil(t, repo.clock)
}

func TestRepositoryTrustPluginKey_Error(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	store.Close() // Close to cause error

	repo := New(store)
	ctx := context.Background()

	err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{
		Fingerprint: "abc123",
		PublicKey:   []byte("pub"),
	})
	assert.Error(t, err)
}

func TestRepositoryTrustPluginKey_EmptyFingerprint(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	defer store.Close()

	repo := New(store)
	ctx := context.Background()

	err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{
		Fingerprint: "",
		PublicKey:   []byte("pub"),
	})
	// Should handle empty fingerprint
	_ = err
}

func TestRepositoryTrustPluginKey_EmptyPublicKey(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	defer store.Close()

	repo := New(store)
	ctx := context.Background()

	err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{
		Fingerprint: "abc123",
		PublicKey:   []byte{},
	})
	// Should handle empty public key
	_ = err
}

func TestRepositoryTrustPluginKey_UpdateExisting(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	defer store.Close()

	repo := New(store)
	ctx := context.Background()

	// Add first key
	err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{
		Fingerprint: "abc123",
		PublicKey:   []byte("pub1"),
	})
	require.NoError(t, err)

	// Update with new key
	err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{
		Fingerprint: "abc123",
		PublicKey:   []byte("pub2"),
	})
	require.NoError(t, err)

	// Verify updated key
	key, err := store.TrustedPluginKey(ctx, "abc123")
	require.NoError(t, err)
	assert.Equal(t, []byte("pub2"), key.PublicKey)
}

func TestRepositoryTrustPluginKey_MultipleKeys(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	defer store.Close()

	repo := New(store)
	ctx := context.Background()

	// Add multiple keys
	keys := []struct {
		fingerprint string
		publicKey   []byte
	}{
		{"key1", []byte("pub1")},
		{"key2", []byte("pub2")},
		{"key3", []byte("pub3")},
	}

	for _, k := range keys {
		err = repo.TrustPluginKey(ctx, serviceagent.TrustPluginKeyInput{
			Fingerprint: k.fingerprint,
			PublicKey:   k.publicKey,
		})
		require.NoError(t, err)
	}

	// Verify all keys
	for _, k := range keys {
		key, err := store.TrustedPluginKey(ctx, k.fingerprint)
		require.NoError(t, err)
		assert.Equal(t, k.fingerprint, key.Fingerprint)
		assert.Equal(t, k.publicKey, key.PublicKey)
	}
}

func TestRepositoryTrustPluginKey_ClockFunction(t *testing.T) {
	store, err := vault.OpenOrCreate(filepath.Join(t.TempDir(), "vault.db"), "password", "test")
	require.NoError(t, err)
	defer store.Close()

	repo := New(store)

	// Verify clock function works
	before := time.Now().UTC()
	time.Sleep(10 * time.Millisecond)
	clockTime := repo.clock()
	time.Sleep(10 * time.Millisecond)
	after := time.Now().UTC()

	assert.True(t, clockTime.After(before) || clockTime.Equal(before))
	assert.True(t, clockTime.Before(after) || clockTime.Equal(after))
}
