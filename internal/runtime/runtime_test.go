package runtime

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())

	require.NotNil(t, manager)
	assert.Equal(t, "1.0.0", manager.version)

	status := manager.Snapshot()
	assert.Equal(t, entity.RunModeIdle, status.Mode)
	assert.Equal(t, entity.ConnectivityStateConnected, status.Connectivity)
	assert.Equal(t, entity.LicenseStateDemo, status.License)
	assert.Equal(t, entity.IntegrityStateTrusted, status.Integrity)
	assert.False(t, status.NewEntriesBlocked)
	assert.Empty(t, status.Banner)
	assert.False(t, status.AnalyticsDegraded)
}

func TestManager_Snapshot(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())

	snapshot1 := manager.Snapshot()
	snapshot2 := manager.Snapshot()

	assert.Equal(t, snapshot1.Mode, snapshot2.Mode)
	assert.Equal(t, snapshot1.Connectivity, snapshot2.Connectivity)
}

func TestManager_Transition(t *testing.T) {
	tests := []struct {
		name string
		mode entity.RunMode
	}{
		{name: "set idle mode", mode: entity.RunModeIdle},
		{name: "set scanning mode", mode: entity.RunModeScanning},
		{name: "set paper trading mode", mode: entity.RunModePaperTrading},
		{name: "set backtesting mode", mode: entity.RunModeBacktesting},
		{name: "set optimizing mode", mode: entity.RunModeOptimizing},
		{name: "set live trading mode", mode: entity.RunModeLiveTrading},
		{name: "set halted mode", mode: entity.RunModeHalted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager("1.0.0", logrus.New())
			before := time.Now().UTC()

			err := manager.Transition(tt.mode, "test transition")
			require.NoError(t, err)

			status := manager.Snapshot()
			assert.Equal(t, tt.mode, status.Mode)
			assert.True(t, status.LastUpdatedAtUTC.After(before) || status.LastUpdatedAtUTC.Equal(before))
		})
	}
}

func TestManager_Transition_HaltedToIdleOnly(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())

	err := manager.Transition(entity.RunModeHalted, "emergency")
	require.NoError(t, err)

	err = manager.Transition(entity.RunModeLiveTrading, "should fail")
	assert.Error(t, err)

	err = manager.Transition(entity.RunModeIdle, "recovery")
	require.NoError(t, err)

	status := manager.Snapshot()
	assert.Equal(t, entity.RunModeIdle, status.Mode)
}

func TestManager_SetConnectivity(t *testing.T) {
	tests := []struct {
		name  string
		state entity.ConnectivityState
	}{
		{name: "set connected", state: entity.ConnectivityStateConnected},
		{name: "set reconnecting", state: entity.ConnectivityStateReconnecting},
		{name: "set network waiting", state: entity.ConnectivityStateNetworkWait},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager("1.0.0", logrus.New())
			before := time.Now().UTC()

			manager.SetConnectivity(tt.state)

			status := manager.Snapshot()
			assert.Equal(t, tt.state, status.Connectivity)
			assert.True(t, status.LastUpdatedAtUTC.After(before) || status.LastUpdatedAtUTC.Equal(before))
		})
	}
}

func TestManager_SetLicense(t *testing.T) {
	tests := []struct {
		name         string
		state        entity.LicenseState
		expectBanner bool
		banner       string
	}{
		{name: "set demo", state: entity.LicenseStateDemo, expectBanner: false},
		{name: "set licensed", state: entity.LicenseStateLicensed, expectBanner: false},
		{name: "set grace", state: entity.LicenseStateGrace, expectBanner: false},
		{name: "set risk only", state: entity.LicenseStateRiskOnly, expectBanner: true, banner: "license expired"},
		{name: "set unlicensed", state: entity.LicenseStateUnlicensed, expectBanner: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager("1.0.0", logrus.New())
			before := time.Now().UTC()

			manager.SetLicense(tt.state)

			status := manager.Snapshot()
			assert.Equal(t, tt.state, status.License)
			if tt.expectBanner {
				assert.Equal(t, tt.banner, status.Banner)
			}
			assert.True(t, status.LastUpdatedAtUTC.After(before) || status.LastUpdatedAtUTC.Equal(before))
		})
	}
}

func TestManager_UpdateNTPDrift(t *testing.T) {
	tests := []struct {
		name                 string
		drift                time.Duration
		expectEntriesBlocked bool
	}{
		{name: "low drift - entries allowed", drift: 100 * time.Millisecond, expectEntriesBlocked: false},
		{name: "medium drift - entries allowed", drift: 300 * time.Millisecond, expectEntriesBlocked: false},
		{name: "high drift - entries blocked", drift: 600 * time.Millisecond, expectEntriesBlocked: true},
		{name: "very high drift - entries blocked", drift: 1000 * time.Millisecond, expectEntriesBlocked: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager("1.0.0", logrus.New())
			before := time.Now().UTC()

			manager.UpdateNTPDrift(tt.drift)

			status := manager.Snapshot()
			assert.Equal(t, tt.drift, status.NTPDrift)
			assert.Equal(t, tt.expectEntriesBlocked, status.NewEntriesBlocked)
			assert.True(t, status.LastUpdatedAtUTC.After(before) || status.LastUpdatedAtUTC.Equal(before))
		})
	}
}

func TestManager_UpdateNTPDrift_Hysteresis(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())

	manager.UpdateNTPDrift(600 * time.Millisecond)
	status := manager.Snapshot()
	assert.True(t, status.NewEntriesBlocked)

	manager.UpdateNTPDrift(300 * time.Millisecond)
	status = manager.Snapshot()
	assert.True(t, status.NewEntriesBlocked)

	manager.UpdateNTPDrift(150 * time.Millisecond)
	status = manager.Snapshot()
	assert.False(t, status.NewEntriesBlocked)
}

func TestManager_SetIntegrity(t *testing.T) {
	tests := []struct {
		name   string
		status entity.IntegrityState
	}{
		{name: "set trusted", status: entity.IntegrityStateTrusted},
		{name: "set degraded", status: entity.IntegrityStateDegraded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager("1.0.0", logrus.New())
			before := time.Now().UTC()

			manager.SetIntegrity(tt.status)

			status := manager.Snapshot()
			assert.Equal(t, tt.status, status.Integrity)
			assert.True(t, status.LastUpdatedAtUTC.After(before) || status.LastUpdatedAtUTC.Equal(before))
		})
	}
}

func TestManager_EnableAnalyticsDegrade(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())
	before := time.Now().UTC()

	status := manager.Snapshot()
	assert.False(t, status.AnalyticsDegraded)

	manager.EnableAnalyticsDegrade()

	status = manager.Snapshot()
	assert.True(t, status.AnalyticsDegraded)
	assert.True(t, status.LastUpdatedAtUTC.After(before) || status.LastUpdatedAtUTC.Equal(before))
}

func TestManager_StartLive(t *testing.T) {
	tests := []struct {
		name               string
		disclaimerAccepted bool
		expectError        bool
		expectedMode       entity.RunMode
	}{
		{name: "start live with disclaimer", disclaimerAccepted: true, expectError: false, expectedMode: entity.RunModeLiveTrading},
		{name: "start live without disclaimer", disclaimerAccepted: false, expectError: true, expectedMode: entity.RunModeIdle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager("1.0.0", logrus.New())

			err := manager.StartLive(tt.disclaimerAccepted)

			if tt.expectError {
				assert.ErrorIs(t, err, ErrLiveTradingRequiresDisclaimer)
			} else {
				assert.NoError(t, err)
			}

			status := manager.Snapshot()
			assert.Equal(t, tt.expectedMode, status.Mode)
		})
	}
}

func TestManager_StartPaper(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())

	err := manager.StartPaper()
	require.NoError(t, err)

	status := manager.Snapshot()
	assert.Equal(t, entity.RunModePaperTrading, status.Mode)
}

func TestManager_StopAll(t *testing.T) {
	manager := NewManager("1.0.0", logrus.New())

	err := manager.StartPaper()
	require.NoError(t, err)
	status := manager.Snapshot()
	assert.Equal(t, entity.RunModePaperTrading, status.Mode)

	err = manager.StopAll()
	require.NoError(t, err)
	status = manager.Snapshot()
	assert.Equal(t, entity.RunModeIdle, status.Mode)
}

func TestUpdateNTPDriftAutoUnblocksEntries(t *testing.T) {
	manager := NewManager("test", logrus.New())
	manager.UpdateNTPDrift(600 * time.Millisecond)
	if !manager.Snapshot().NewEntriesBlocked {
		t.Fatal("expected entries to be blocked at high drift")
	}
	manager.UpdateNTPDrift(150 * time.Millisecond)
	if manager.Snapshot().NewEntriesBlocked {
		t.Fatal("expected entries to be unblocked after drift recovery")
	}
}
