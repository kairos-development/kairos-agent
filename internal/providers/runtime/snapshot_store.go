package runtimeprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

var errRuntimeSnapshotNotFound = errors.New("runtime snapshot not found")

type runtimeSnapshot struct {
	Mode              string `json:"mode"`
	Connectivity      string `json:"connectivity"`
	License           string `json:"license"`
	Integrity         string `json:"integrity"`
	NTPDriftNanos     int64  `json:"ntp_drift_nanos"`
	NewEntriesBlocked bool   `json:"new_entries_blocked"`
	HaltReason        string `json:"halt_reason"`
	HaltedAtUTC       string `json:"halted_at_utc,omitempty"`
	Banner            string `json:"banner"`
	AnalyticsDegraded bool   `json:"analytics_degraded"`
	LastUpdatedAtUTC  string `json:"last_updated_at_utc"`
}

// SnapshotStore persists runtime status snapshots for crash-safe diagnostics.
type SnapshotStore struct {
	mu   sync.Mutex
	path string
}

// NewSnapshotStore creates a file-backed runtime snapshot store.
func NewSnapshotStore(path string) *SnapshotStore {
	return &SnapshotStore{path: path}
}

// Load reads the latest runtime status snapshot.
func (s *SnapshotStore) Load(ctx context.Context) (entity.RuntimeStatus, bool, error) {
	if s == nil || s.path == "" {
		return entity.RuntimeStatus{}, false, nil
	}
	if err := ctx.Err(); err != nil {
		return entity.RuntimeStatus{}, false, err
	}

	body, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return entity.RuntimeStatus{}, false, nil
	}
	if err != nil {
		return entity.RuntimeStatus{}, false, fmt.Errorf("read runtime snapshot: %w", err)
	}

	var snapshot runtimeSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return entity.RuntimeStatus{}, false, fmt.Errorf("decode runtime snapshot: %w", err)
	}

	status, err := snapshot.toDomain()
	if err != nil {
		return entity.RuntimeStatus{}, false, err
	}
	return status, true, nil
}

// Save writes a runtime status snapshot atomically.
func (s *SnapshotStore) Save(ctx context.Context, status entity.RuntimeStatus) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create runtime snapshot dir: %w", err)
	}

	body, err := json.MarshalIndent(snapshotFromDomain(status), "", "  ")
	if err != nil {
		return fmt.Errorf("encode runtime snapshot: %w", err)
	}
	body = append(body, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".runtime-status-*.tmp")
	if err != nil {
		return fmt.Errorf("create runtime snapshot temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write runtime snapshot temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync runtime snapshot temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close runtime snapshot temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("chmod runtime snapshot temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace runtime snapshot: %w", err)
	}
	return nil
}

func snapshotFromDomain(status entity.RuntimeStatus) runtimeSnapshot {
	snapshot := runtimeSnapshot{
		Mode:              string(status.Mode),
		Connectivity:      string(status.Connectivity),
		License:           string(status.License),
		Integrity:         string(status.Integrity),
		NTPDriftNanos:     int64(status.NTPDrift),
		NewEntriesBlocked: status.NewEntriesBlocked,
		HaltReason:        status.HaltReason,
		Banner:            status.Banner,
		AnalyticsDegraded: status.AnalyticsDegraded,
		LastUpdatedAtUTC:  formatSnapshotTime(status.LastUpdatedAtUTC),
	}
	if status.HaltedAtUTC != nil && !status.HaltedAtUTC.IsZero() {
		snapshot.HaltedAtUTC = formatSnapshotTime(*status.HaltedAtUTC)
	}
	return snapshot
}

func (s runtimeSnapshot) toDomain() (entity.RuntimeStatus, error) {
	lastUpdated, err := parseSnapshotTime("last_updated_at_utc", s.LastUpdatedAtUTC)
	if err != nil {
		return entity.RuntimeStatus{}, err
	}

	var haltedAt *time.Time
	if s.HaltedAtUTC != "" {
		parsed, err := parseSnapshotTime("halted_at_utc", s.HaltedAtUTC)
		if err != nil {
			return entity.RuntimeStatus{}, err
		}
		haltedAt = &parsed
	}

	return entity.RuntimeStatus{
		Mode:              entity.RunMode(s.Mode),
		Connectivity:      entity.ConnectivityState(s.Connectivity),
		License:           entity.LicenseState(s.License),
		Integrity:         entity.IntegrityState(s.Integrity),
		NTPDrift:          time.Duration(s.NTPDriftNanos),
		NewEntriesBlocked: s.NewEntriesBlocked,
		HaltReason:        s.HaltReason,
		HaltedAtUTC:       haltedAt,
		Banner:            s.Banner,
		AnalyticsDegraded: s.AnalyticsDegraded,
		LastUpdatedAtUTC:  lastUpdated,
	}, nil
}

func formatSnapshotTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseSnapshotTime(field string, raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse runtime snapshot %s=%q: %w", field, raw, err)
	}
	return parsed.UTC(), nil
}
