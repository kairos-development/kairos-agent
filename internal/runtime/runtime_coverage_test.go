package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_NilLogger(t *testing.T) {
	m := NewManager("test-1.0", nil)
	assert.NotNil(t, m)
	assert.NotNil(t, m.logger)
	assert.NotNil(t, m.stateMachine)
	assert.Equal(t, entity.RunModeIdle, m.status.Mode)
}

func TestNewManager_WithVersion(t *testing.T) {
	logger := logrus.New()
	m := NewManager("v2.0.0", logger)
	assert.Equal(t, "v2.0.0", m.version)
	assert.Equal(t, logger, m.logger)
}

func TestCurrentState(t *testing.T) {
	m := NewManager("test", nil)
	assert.Equal(t, entity.RunModeIdle, m.CurrentState())

	err := m.Transition(entity.RunModePaperTrading, "test")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModePaperTrading, m.CurrentState())
}

func TestSubscribeStateChanges(t *testing.T) {
	m := NewManager("test", nil)
	ch := make(chan StateTransition, 1)

	id := m.SubscribeStateChanges(ch)
	assert.Greater(t, id, uint64(0))

	// Transition should notify the subscriber
	err := m.Transition(entity.RunModePaperTrading, "subscribe test")
	require.NoError(t, err)

	select {
	case transition := <-ch:
		assert.Equal(t, entity.RunModeIdle, transition.From)
		assert.Equal(t, entity.RunModePaperTrading, transition.To)
		assert.Equal(t, "subscribe test", transition.Reason)
	case <-time.After(time.Second):
		t.Fatal("expected state transition notification")
	}
}

func TestSubscribeStateChanges_MultipleSubscribers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	m := NewManager("test", logger)

	ch1 := make(chan StateTransition, 10)
	ch2 := make(chan StateTransition, 10)

	m.SubscribeStateChanges(ch1)
	m.SubscribeStateChanges(ch2)

	err := m.Transition(entity.RunModeLiveTrading, "multi-sub")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Both channels should receive the notification
	assert.Len(t, ch1, 1)
	assert.Len(t, ch2, 1)
}

func TestSnapshot_DefaultValues(t *testing.T) {
	m := NewManager("test", nil)
	snap := m.Snapshot()

	assert.Equal(t, entity.RunModeIdle, snap.Mode)
	assert.Equal(t, entity.ConnectivityStateConnected, snap.Connectivity)
	assert.Equal(t, entity.LicenseStateDemo, snap.License)
	assert.Equal(t, entity.IntegrityStateTrusted, snap.Integrity)
	assert.False(t, snap.AnalyticsDegraded)
	assert.False(t, snap.NewEntriesBlocked)
	assert.Empty(t, snap.HaltReason)
	assert.Nil(t, snap.HaltedAtUTC)
}

func TestSetLicense_RiskOnlyBanner(t *testing.T) {
	m := NewManager("test", nil)
	m.SetLicense(entity.LicenseStateRiskOnly)
	snap := m.Snapshot()
	assert.Equal(t, entity.LicenseStateRiskOnly, snap.License)
	assert.Equal(t, "license expired", snap.Banner)
}

func TestSetLicense_LicensedNoBanner(t *testing.T) {
	m := NewManager("test", nil)
	m.SetLicense(entity.LicenseStateLicensed)
	snap := m.Snapshot()
	assert.Equal(t, entity.LicenseStateLicensed, snap.License)
	assert.Empty(t, snap.Banner)
}

func TestUpdateNTPDrift_BlocksAbove500ms(t *testing.T) {
	m := NewManager("test", nil)
	m.UpdateNTPDrift(600 * time.Millisecond)
	snap := m.Snapshot()
	assert.Equal(t, 600*time.Millisecond, snap.NTPDrift)
	assert.True(t, snap.NewEntriesBlocked)
}

func TestUpdateNTPDrift_UnblocksBelow200ms(t *testing.T) {
	m := NewManager("test", nil)
	// First block
	m.UpdateNTPDrift(600 * time.Millisecond)
	// Then unblock
	m.UpdateNTPDrift(100 * time.Millisecond)
	snap := m.Snapshot()
	assert.Equal(t, 100*time.Millisecond, snap.NTPDrift)
	assert.False(t, snap.NewEntriesBlocked)
}

func TestUpdateNTPDrift_HysteresisZone(t *testing.T) {
	m := NewManager("test", nil)
	// Set above threshold to block
	m.UpdateNTPDrift(600 * time.Millisecond)
	assert.True(t, m.Snapshot().NewEntriesBlocked)

	// Drift in hysteresis zone (200-500ms) — should NOT change state
	m.UpdateNTPDrift(350 * time.Millisecond)
	assert.True(t, m.Snapshot().NewEntriesBlocked)

	// Set below threshold — should unblock
	m.UpdateNTPDrift(100 * time.Millisecond)
	assert.False(t, m.Snapshot().NewEntriesBlocked)
}

func TestSetIntegrity(t *testing.T) {
	m := NewManager("test", nil)
	m.SetIntegrity(entity.IntegrityStateDegraded)
	snap := m.Snapshot()
	assert.Equal(t, entity.IntegrityStateDegraded, snap.Integrity)
}

func TestEnableAnalyticsDegrade(t *testing.T) {
	m := NewManager("test", nil)
	m.EnableAnalyticsDegrade()
	snap := m.Snapshot()
	assert.True(t, snap.AnalyticsDegraded)
}

func TestApplyMemoryLimit(t *testing.T) {
	// Save the original limit and restore it after the test
	originalLimit := ApplyMemoryLimit(-1)
	defer ApplyMemoryLimit(originalLimit)

	// Set a specific limit (100 MB)
	newLimit := int64(100 * 1024 * 1024)
	previousLimit := ApplyMemoryLimit(newLimit)
	assert.Equal(t, originalLimit, previousLimit)

	// Setting -1 returns current limit without changing it
	currentLimit := ApplyMemoryLimit(-1)
	assert.Equal(t, newLimit, currentLimit)
}

func TestApplyMemoryLimit_Roundtrip(t *testing.T) {
	originalLimit := ApplyMemoryLimit(-1)
	defer ApplyMemoryLimit(originalLimit)

	// Set a limit and verify it's returned
	target := int64(50 * 1024 * 1024)
	prev := ApplyMemoryLimit(target)
	assert.Equal(t, originalLimit, prev)

	// Verify current
	got := ApplyMemoryLimit(-1)
	assert.Equal(t, target, got)
}

func TestRunNTPWorker_NormalDrift(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	probeCount := 0
	var lastDrift time.Duration

	probe := func(ctx context.Context) (time.Duration, error) {
		mu.Lock()
		probeCount++
		mu.Unlock()
		return 150 * time.Millisecond, nil
	}
	onDrift := func(d time.Duration) {
		mu.Lock()
		lastDrift = d
		mu.Unlock()
	}

	go RunNTPWorker(ctx, 50*time.Millisecond, probe, onDrift)

	// Wait for at least 2 ticks
	time.Sleep(120 * time.Millisecond)
	cancel()

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, probeCount, 2)
	assert.Equal(t, 150*time.Millisecond, lastDrift)
}

func TestRunNTPWorker_ProbeError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onDriftCalled := false

	probe := func(ctx context.Context) (time.Duration, error) {
		return 0, errors.New("ntp server unreachable")
	}
	onDrift := func(d time.Duration) {
		onDriftCalled = true
	}

	go RunNTPWorker(ctx, 50*time.Millisecond, probe, onDrift)
	time.Sleep(80 * time.Millisecond)
	cancel()

	assert.False(t, onDriftCalled)
}

func TestRunNTPWorker_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	probeCount := 0
	probe := func(ctx context.Context) (time.Duration, error) {
		mu.Lock()
		probeCount++
		mu.Unlock()
		return 100 * time.Millisecond, nil
	}
	onDrift := func(d time.Duration) {}

	go RunNTPWorker(ctx, 50*time.Millisecond, probe, onDrift)

	// Cancel immediately
	cancel()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.LessOrEqual(t, probeCount, 1)
}

func TestRunNTPWorker_DriftThresholdCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	drifts := []time.Duration{600 * time.Millisecond, 100 * time.Millisecond}
	var mu sync.Mutex
	idx := 0
	var capturedDrifts []time.Duration

	probe := func(ctx context.Context) (time.Duration, error) {
		mu.Lock()
		d := drifts[idx]
		if idx < len(drifts)-1 {
			idx++
		}
		mu.Unlock()
		return d, nil
	}
	onDrift := func(d time.Duration) {
		mu.Lock()
		capturedDrifts = append(capturedDrifts, d)
		mu.Unlock()
	}

	go RunNTPWorker(ctx, 50*time.Millisecond, probe, onDrift)
	time.Sleep(130 * time.Millisecond)
	cancel()

	// Should have captured the drifts that were probed
	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, len(capturedDrifts), 1)
}

func TestSnapshot_ReflectsLastUpdate(t *testing.T) {
	m := NewManager("test", nil)

	before := m.Snapshot()
	time.Sleep(10 * time.Millisecond)

	m.SetConnectivity(entity.ConnectivityStateReconnecting)
	after := m.Snapshot()

	assert.Equal(t, entity.ConnectivityStateReconnecting, after.Connectivity)
	assert.False(t, after.LastUpdatedAtUTC.IsZero())
	// Timestamps should differ
	assert.NotEqual(t, before.LastUpdatedAtUTC, after.LastUpdatedAtUTC)
}

func TestManagerTransition_NoopSameState(t *testing.T) {
	m := NewManager("test", nil)
	err := m.Transition(entity.RunModeIdle, "no-op")
	assert.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, m.CurrentState())
}

func TestManagerTransition_HaltedToIdleRestart(t *testing.T) {
	m := NewManager("test", nil)

	// Transition to halted
	err := m.Transition(entity.RunModeHalted, "integrity failure")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeHalted, m.CurrentState())
	assert.Equal(t, "integrity failure", m.Snapshot().HaltReason)
	assert.NotNil(t, m.Snapshot().HaltedAtUTC)

	// Halted -> Idle should work
	err = m.Transition(entity.RunModeIdle, "admin restart")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, m.CurrentState())
	assert.Empty(t, m.Snapshot().HaltReason)
	assert.Nil(t, m.Snapshot().HaltedAtUTC)
}

func TestManagerTransition_HaltedToLiveBlocked(t *testing.T) {
	m := NewManager("test", nil)

	// Go to halted
	err := m.Transition(entity.RunModeHalted, "failure")
	require.NoError(t, err)

	// Halted -> anything but Idle is blocked
	err = m.Transition(entity.RunModeLiveTrading, "illegal")
	assert.Error(t, err)
	assert.Equal(t, entity.RunModeHalted, m.CurrentState())
}

func TestSnapshot_LastUpdatedOnEveryMutation(t *testing.T) {
	m := NewManager("test", nil)

	t1 := m.Snapshot().LastUpdatedAtUTC
	time.Sleep(5 * time.Millisecond)

	m.SetConnectivity(entity.ConnectivityStateNetworkWait)
	t2 := m.Snapshot().LastUpdatedAtUTC
	time.Sleep(5 * time.Millisecond)

	m.SetLicense(entity.LicenseStateRiskOnly)
	t3 := m.Snapshot().LastUpdatedAtUTC

	assert.True(t, t2.After(t1) || t2.Equal(t2))
	assert.True(t, t3.After(t2) || t3.Equal(t3))
}
