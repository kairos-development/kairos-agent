package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/beevik/ntp"
)

// NTPSyncWorker monitors system time drift against NTP servers.
type NTPSyncWorker struct {
	maxDrift time.Duration
	servers  []string
	lastSync time.Time
	drift    time.Duration
}

// NewNTPSyncWorker creates a new NTP sync worker.
func NewNTPSyncWorker(maxDrift time.Duration) *NTPSyncWorker {
	return &NTPSyncWorker{
		maxDrift: maxDrift,
		servers: []string{
			"pool.ntp.org",
			"time.nist.gov",
			"time.google.com",
		},
	}
}

// Check performs an NTP time check and returns the drift.
func (w *NTPSyncWorker) Check(ctx context.Context) (time.Duration, error) {
	var lastErr error

	// Try each NTP server
	for _, server := range w.servers {
		response, err := ntp.QueryWithOptions(server, ntp.QueryOptions{
			Timeout: 5 * time.Second,
		})
		if err != nil {
			lastErr = err
			continue
		}

		// Calculate drift
		drift := response.ClockOffset
		if drift < 0 {
			drift = -drift
		}

		w.drift = drift
		w.lastSync = time.Now().UTC()

		return drift, nil
	}

	return 0, fmt.Errorf("NTP sync failed: %w", lastErr)
}

// Drift returns the last measured time drift.
func (w *NTPSyncWorker) Drift() time.Duration {
	return w.drift
}

// LastSync returns when the last successful sync occurred.
func (w *NTPSyncWorker) LastSync() time.Time {
	return w.lastSync
}

// IsHealthy returns true if drift is within acceptable limits.
func (w *NTPSyncWorker) IsHealthy() bool {
	return w.drift <= w.maxDrift
}
