package runtimeprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotStore_SaveLoadRoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime-status.json")
	store := NewSnapshotStore(path)
	haltedAt := time.Now().UTC().Truncate(time.Nanosecond)

	status := entity.RuntimeStatus{
		Mode:              entity.RunModeHalted,
		Connectivity:      entity.ConnectivityStateNetworkWait,
		License:           entity.LicenseStateGrace,
		Integrity:         entity.IntegrityStateTrusted,
		NTPDrift:          42 * time.Millisecond,
		NewEntriesBlocked: true,
		HaltReason:        "stream gap",
		HaltedAtUTC:       &haltedAt,
		Banner:            "halted: stream gap",
		AnalyticsDegraded: true,
		LastUpdatedAtUTC:  haltedAt,
	}

	require.NoError(t, store.Save(ctx, status))

	loaded, found, err := store.Load(ctx)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, status.Mode, loaded.Mode)
	assert.Equal(t, status.Connectivity, loaded.Connectivity)
	assert.Equal(t, status.License, loaded.License)
	assert.Equal(t, status.Integrity, loaded.Integrity)
	assert.Equal(t, status.NTPDrift, loaded.NTPDrift)
	assert.True(t, loaded.NewEntriesBlocked)
	assert.Equal(t, status.HaltReason, loaded.HaltReason)
	require.NotNil(t, loaded.HaltedAtUTC)
	assert.Equal(t, haltedAt, *loaded.HaltedAtUTC)
	assert.Equal(t, status.Banner, loaded.Banner)
	assert.True(t, loaded.AnalyticsDegraded)
	assert.Equal(t, status.LastUpdatedAtUTC, loaded.LastUpdatedAtUTC)
}

func TestSnapshotStore_LoadMissing(t *testing.T) {
	loaded, found, err := NewSnapshotStore(filepath.Join(t.TempDir(), "missing.json")).Load(context.Background())
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, loaded.Mode)
}

func TestSnapshotStore_LoadCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime-status.json")
	require.NoError(t, os.WriteFile(path, []byte("{bad json"), 0o600))

	_, found, err := NewSnapshotStore(path).Load(context.Background())
	assert.False(t, found)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode runtime snapshot")
}

func TestSnapshotStore_SaveHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := NewSnapshotStore(filepath.Join(t.TempDir(), "runtime-status.json")).Save(ctx, entity.RuntimeStatus{})
	require.ErrorIs(t, err, context.Canceled)
}
