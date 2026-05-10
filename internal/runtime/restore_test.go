package runtime

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_RestoreSafetySnapshot_PreservesHaltedReason(t *testing.T) {
	manager := NewManager("test", logrus.New())
	haltedAt := time.Now().UTC().Add(-time.Minute)

	manager.RestoreSafetySnapshot(Status{
		Mode:             entity.RunModeHalted,
		Connectivity:     entity.ConnectivityStateNetworkWait,
		License:          entity.LicenseStateLicensed,
		Integrity:        entity.IntegrityStateTrusted,
		HaltReason:       "stream gap",
		HaltedAtUTC:      &haltedAt,
		LastUpdatedAtUTC: haltedAt,
	})

	status := manager.Snapshot()
	assert.Equal(t, entity.RunModeHalted, status.Mode)
	assert.Equal(t, "stream gap", status.HaltReason)
	require.NotNil(t, status.HaltedAtUTC)
	assert.Equal(t, haltedAt, *status.HaltedAtUTC)
	assert.True(t, status.NewEntriesBlocked)
	assert.Equal(t, "halted: stream gap", status.Banner)
	assert.Equal(t, entity.RunModeHalted, manager.CurrentState())
}

func TestManager_RestoreSafetySnapshot_ActiveModeBecomesHalted(t *testing.T) {
	manager := NewManager("test", logrus.New())

	manager.RestoreSafetySnapshot(Status{
		Mode:             entity.RunModeLiveTrading,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateLicensed,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC().Add(-time.Hour),
	})

	status := manager.Snapshot()
	assert.Equal(t, entity.RunModeHalted, status.Mode)
	assert.Equal(t, "unclean restart from live_trading", status.HaltReason)
	require.NotNil(t, status.HaltedAtUTC)
	assert.True(t, status.NewEntriesBlocked)
	assert.Equal(t, entity.RunModeHalted, manager.CurrentState())
}

func TestManager_RestoreSafetySnapshot_DefaultsMissingFields(t *testing.T) {
	manager := NewManager("test", logrus.New())

	manager.RestoreSafetySnapshot(Status{})

	status := manager.Snapshot()
	assert.Equal(t, entity.RunModeIdle, status.Mode)
	assert.Equal(t, entity.ConnectivityStateConnected, status.Connectivity)
	assert.Equal(t, entity.LicenseStateDemo, status.License)
	assert.Equal(t, entity.IntegrityStateTrusted, status.Integrity)
	assert.NotZero(t, status.LastUpdatedAtUTC)
}
