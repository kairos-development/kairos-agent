package runtimeprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryMapsRuntimeState(t *testing.T) {
	manager := runtime.NewManager("test", logrus.New())
	repo := New(manager)
	ctx := context.Background()

	require.NoError(t, repo.Transition(ctx, entity.RunModeLiveTrading, "test"))
	require.NoError(t, repo.SetLicense(ctx, entity.LicenseStateLicensed))
	require.NoError(t, repo.SetConnectivity(ctx, entity.ConnectivityStateNetworkWait))

	status, err := repo.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeLiveTrading, status.Mode)
	assert.Equal(t, entity.LicenseStateLicensed, status.License)
	assert.Equal(t, entity.ConnectivityStateNetworkWait, status.Connectivity)
}

func TestRepositoryPersistsAndRestoresHaltedSnapshot(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime-status.json")
	store := NewSnapshotStore(path)

	manager := runtime.NewManager("test", logrus.New())
	repo := New(manager, WithSnapshotStore(store))

	require.NoError(t, repo.Transition(ctx, entity.RunModeHalted, "stream gap"))

	restoredManager := runtime.NewManager("test", logrus.New())
	restoredRepo := New(restoredManager, WithSnapshotStore(store))

	status, err := restoredRepo.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeHalted, status.Mode)
	assert.Equal(t, "stream gap", status.HaltReason)
	assert.True(t, status.NewEntriesBlocked)
	assert.NotNil(t, status.HaltedAtUTC)
}

func TestRepositoryRestoresActiveSnapshotAsHalted(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime-status.json")
	store := NewSnapshotStore(path)
	now := time.Now().UTC()

	require.NoError(t, store.Save(ctx, entity.RuntimeStatus{
		Mode:             entity.RunModeLiveTrading,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateLicensed,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: now,
	}))

	manager := runtime.NewManager("test", logrus.New())
	repo := New(manager, WithSnapshotStore(store))

	status, err := repo.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeHalted, status.Mode)
	assert.Equal(t, "unclean restart from live_trading", status.HaltReason)
	assert.True(t, status.NewEntriesBlocked)
}

func TestRepositoryCorruptSnapshotStartsHalted(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runtime-status.json")
	require.NoError(t, os.WriteFile(path, []byte("{bad json"), 0o600))

	manager := runtime.NewManager("test", logrus.New())
	repo := New(manager, WithSnapshotStore(NewSnapshotStore(path)))

	status, err := repo.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeHalted, status.Mode)
	assert.Contains(t, status.HaltReason, "runtime snapshot restore failed")
	assert.True(t, status.NewEntriesBlocked)
}
