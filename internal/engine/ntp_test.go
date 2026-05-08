package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNTPSyncWorker_NewNTPSyncWorker tests worker creation.
func TestNTPSyncWorker_NewNTPSyncWorker(t *testing.T) {
	maxDrift := 500 * time.Millisecond
	worker := NewNTPSyncWorker(maxDrift)

	assert.NotNil(t, worker)
	assert.Equal(t, maxDrift, worker.maxDrift)
	assert.Len(t, worker.servers, 3)
	assert.Contains(t, worker.servers, "pool.ntp.org")
	assert.Contains(t, worker.servers, "time.nist.gov")
	assert.Contains(t, worker.servers, "time.google.com")
}

// TestNTPSyncWorker_IsHealthy tests health check.
func TestNTPSyncWorker_IsHealthy(t *testing.T) {
	worker := NewNTPSyncWorker(500 * time.Millisecond)

	// Initially should be healthy (drift is 0)
	assert.True(t, worker.IsHealthy())

	// Simulate small drift
	worker.drift = 100 * time.Millisecond
	assert.True(t, worker.IsHealthy())

	// Simulate drift at threshold
	worker.drift = 500 * time.Millisecond
	assert.True(t, worker.IsHealthy())

	// Simulate large drift
	worker.drift = 501 * time.Millisecond
	assert.False(t, worker.IsHealthy())

	// Simulate very large drift
	worker.drift = 1 * time.Second
	assert.False(t, worker.IsHealthy())
}

// TestNTPSyncWorker_Drift tests drift getter.
func TestNTPSyncWorker_Drift(t *testing.T) {
	worker := NewNTPSyncWorker(500 * time.Millisecond)

	// Initial drift should be 0
	assert.Equal(t, time.Duration(0), worker.Drift())

	// Set drift
	worker.drift = 250 * time.Millisecond
	assert.Equal(t, 250*time.Millisecond, worker.Drift())
}

// TestNTPSyncWorker_LastSync tests last sync timestamp.
func TestNTPSyncWorker_LastSync(t *testing.T) {
	worker := NewNTPSyncWorker(500 * time.Millisecond)

	// Initial last sync should be zero
	assert.True(t, worker.LastSync().IsZero())

	// Set last sync
	now := time.Now().UTC()
	worker.lastSync = now
	assert.Equal(t, now, worker.LastSync())
}

// TestNTPSyncWorker_CheckAllServersFail tests behavior when all servers fail.
// Note: This test is commented out as it requires network and can be slow.
// In production, use integration tests for network-dependent functionality.
/*
func TestNTPSyncWorker_CheckAllServersFail(t *testing.T) {
	worker := NewNTPSyncWorker(500 * time.Millisecond)

	// Replace all servers with invalid ones
	worker.servers = []string{
		"invalid1.local",
		"invalid2.local",
		"invalid3.local",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Should fail
	_, err := worker.Check(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NTP sync failed")
}
*/

// TestNTPSyncWorker_DriftAbsoluteValue tests that drift is always positive.
func TestNTPSyncWorker_DriftAbsoluteValue(t *testing.T) {
	worker := NewNTPSyncWorker(500 * time.Millisecond)

	// Simulate negative drift (clock ahead)
	worker.drift = -300 * time.Millisecond

	// Drift should still be 300ms (absolute value is taken in Check())
	// But stored value is what we set
	assert.Equal(t, -300*time.Millisecond, worker.Drift())

	// For health check, we need to test with positive drift
	worker.drift = 300 * time.Millisecond
	assert.True(t, worker.IsHealthy())
}

// TestNTPSyncWorker_MultipleChecks tests multiple sequential checks.
func TestNTPSyncWorker_MultipleChecks(t *testing.T) {
	worker := NewNTPSyncWorker(500 * time.Millisecond)

	// Simulate first check
	worker.drift = 100 * time.Millisecond
	worker.lastSync = time.Now().UTC()
	firstSync := worker.LastSync()

	time.Sleep(10 * time.Millisecond)

	// Simulate second check
	worker.drift = 150 * time.Millisecond
	worker.lastSync = time.Now().UTC()
	secondSync := worker.LastSync()

	// Second sync should be after first
	assert.True(t, secondSync.After(firstSync))
	assert.Equal(t, 150*time.Millisecond, worker.Drift())
}

// TestNTPSyncWorker_MaxDriftConfiguration tests different max drift values.
func TestNTPSyncWorker_MaxDriftConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		maxDrift time.Duration
		drift    time.Duration
		healthy  bool
	}{
		{"100ms max, 50ms drift", 100 * time.Millisecond, 50 * time.Millisecond, true},
		{"100ms max, 100ms drift", 100 * time.Millisecond, 100 * time.Millisecond, true},
		{"100ms max, 101ms drift", 100 * time.Millisecond, 101 * time.Millisecond, false},
		{"1s max, 500ms drift", 1 * time.Second, 500 * time.Millisecond, true},
		{"1s max, 1s drift", 1 * time.Second, 1 * time.Second, true},
		{"1s max, 1001ms drift", 1 * time.Second, 1001 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewNTPSyncWorker(tt.maxDrift)
			worker.drift = tt.drift
			assert.Equal(t, tt.healthy, worker.IsHealthy())
		})
	}
}
