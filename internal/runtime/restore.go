package runtime

import (
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

// RestoreSafetySnapshot restores persisted runtime state using safe restart semantics.
func (m *Manager) RestoreSafetySnapshot(snapshot Status) {
	now := time.Now().UTC()
	restored := snapshot

	switch snapshot.Mode {
	case entity.RunModeHalted:
		if restored.HaltReason == "" {
			restored.HaltReason = "restored halted runtime snapshot"
		}
		if restored.HaltedAtUTC == nil || restored.HaltedAtUTC.IsZero() {
			haltedAt := fallbackTime(snapshot.LastUpdatedAtUTC, now)
			restored.HaltedAtUTC = &haltedAt
		}
		restored.NewEntriesBlocked = true
		restored.Banner = haltedBanner(restored.HaltReason)

	case entity.RunModeLiveTrading, entity.RunModePaperTrading, entity.RunModeScanning, entity.RunModeBacktesting, entity.RunModeOptimizing:
		reason := fmt.Sprintf("unclean restart from %s", snapshot.Mode)
		haltedAt := now
		restored.Mode = entity.RunModeHalted
		restored.HaltReason = reason
		restored.HaltedAtUTC = &haltedAt
		restored.NewEntriesBlocked = true
		restored.Banner = haltedBanner(reason)
		restored.LastUpdatedAtUTC = now

	default:
		restored.Mode = entity.RunModeIdle
		restored.HaltReason = ""
		restored.HaltedAtUTC = nil
		restored.Banner = ""
		if restored.LastUpdatedAtUTC.IsZero() {
			restored.LastUpdatedAtUTC = now
		}
	}

	if restored.Connectivity == "" {
		restored.Connectivity = entity.ConnectivityStateConnected
	}
	if restored.License == "" {
		restored.License = entity.LicenseStateDemo
	}
	if restored.Integrity == "" {
		restored.Integrity = entity.IntegrityStateTrusted
	}
	if restored.LastUpdatedAtUTC.IsZero() {
		restored.LastUpdatedAtUTC = now
	}

	m.mu.Lock()
	m.status = cloneStatus(restored)
	m.mu.Unlock()
	m.stateMachine.Restore(restored.Mode, restored.LastUpdatedAtUTC)
}

func cloneStatus(status Status) Status {
	clone := status
	if status.HaltedAtUTC != nil {
		haltedAt := *status.HaltedAtUTC
		clone.HaltedAtUTC = &haltedAt
	}
	return clone
}

func fallbackTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback.UTC()
	}
	return value.UTC()
}

func haltedBanner(reason string) string {
	if reason == "" {
		return "halted"
	}
	return "halted: " + reason
}
