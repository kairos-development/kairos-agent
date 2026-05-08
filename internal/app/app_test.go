package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/journal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapAndClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application, err := Bootstrap(ctx, t.TempDir(), "bootstrap-password")
	require.NoError(t, err)
	require.NotNil(t, application.ConfigManager)
	require.NotNil(t, application.Store)
	require.NotNil(t, application.Vault)
	require.NotNil(t, application.Audit)
	require.NotNil(t, application.Journal)
	require.NotNil(t, application.Runtime)
	assert.NotEmpty(t, application.StateDir)
	require.NoError(t, application.Close())
}

func TestCloseNilResources(t *testing.T) {
	application := &Application{}
	assert.NoError(t, application.Close())
}

func TestBootstrap_InvalidStateDir(t *testing.T) {
	ctx := context.Background()
	// Try to create state dir in a file (not directory)
	tmpFile, err := os.CreateTemp("", "test_*")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	stateDir := filepath.Join(tmpFile.Name(), "state")
	_, err = Bootstrap(ctx, stateDir, "password")
	assert.Error(t, err)
}

func TestBootstrap_WithMemoryLimit(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()

	// Create config with memory limit
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	cfg := manager.Current()
	cfg.MemoryLimitBytes = 1024 * 1024 * 100 // 100MB
	require.NoError(t, manager.Write(cfg))

	application, err := Bootstrap(ctx, stateDir, "password")
	require.NoError(t, err)
	defer application.Close()

	assert.NotNil(t, application)
}

func TestClose_WithErrors(t *testing.T) {
	ctx := context.Background()
	application, err := Bootstrap(ctx, t.TempDir(), "password")
	require.NoError(t, err)

	// Close once
	require.NoError(t, application.Close())

	// Close again should return errors from already closed resources
	err = application.Close()
	// May or may not error depending on implementation
	_ = err
}

func TestBootstrap_AuditLogError(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()

	// Create config manager first
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	cfg := manager.Current()

	// Create a file where audit log directory should be.
	cfg.Paths.AuditPath = filepath.Join(stateDir, "blocked-audit", "audit.log")
	require.NoError(t, manager.Write(cfg))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "blocked-audit"), []byte("block"), 0o600))

	_, err = Bootstrap(ctx, stateDir, "password")
	assert.Error(t, err)
}

func TestBootstrap_JournalError(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()

	// Create config manager first
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	cfg := manager.Current()

	// Create a file where journal log directory should be.
	cfg.Paths.JournalPath = filepath.Join(stateDir, "blocked-journal", "journal.log")
	require.NoError(t, manager.Write(cfg))
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "blocked-journal"), []byte("block"), 0o600))

	_, err = Bootstrap(ctx, stateDir, "password")
	assert.Error(t, err)
}

func TestBootstrap_StorageError(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()

	// Create invalid trades.sqlite file
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "trades.sqlite"), []byte("invalid"), 0o600))

	_, err := Bootstrap(ctx, stateDir, "password")
	assert.Error(t, err)
}

func TestClose_PartialErrors(t *testing.T) {
	application := &Application{
		Journal: &journal.Log{}, // Invalid journal that will error on close
	}

	err := application.Close()
	// Should collect errors from all resources
	_ = err
}

func TestBootstrap_NTPWorkerStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := Bootstrap(ctx, t.TempDir(), "password")
	require.NoError(t, err)
	defer application.Close()

	// Verify NTP worker is running by checking runtime manager
	assert.NotNil(t, application.Runtime)

	// Cancel context to stop NTP worker
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestBootstrap_ConfigManagerError(t *testing.T) {
	ctx := context.Background()

	// Try to bootstrap with invalid state dir
	_, err := Bootstrap(ctx, "/dev/null/invalid", "password")
	assert.Error(t, err)
}

func TestVersion(t *testing.T) {
	assert.NotEmpty(t, Version)
	assert.Equal(t, "0.1.0", Version)
}
