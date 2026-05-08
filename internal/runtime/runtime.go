package runtime

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/sirupsen/logrus"
)

var (
	// ErrLiveTradingRequiresDisclaimer reports a missing financial risk disclaimer acceptance.
	ErrLiveTradingRequiresDisclaimer = errors.New("live trading requires accepted financial risk disclaimer")
)

// Manager owns mutable runtime state per instruction 8.3.
type Manager struct {
	mu           sync.RWMutex
	stateMachine *StateMachine
	status       Status
	version      string
	logger       *logrus.Logger
}

// NewManager constructs a runtime manager with default values.
func NewManager(version string, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}
	return &Manager{
		stateMachine: NewStateMachine(logger),
		status:       Status{Mode: entity.RunModeIdle, Connectivity: entity.ConnectivityStateConnected, License: entity.LicenseStateDemo, Integrity: entity.IntegrityStateTrusted, LastUpdatedAtUTC: time.Now().UTC()},
		version:      version,
		logger:       logger,
	}
}

// Snapshot returns a copy of the runtime status.
func (m *Manager) Snapshot() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Transition attempts a state change through the state machine.
func (m *Manager) Transition(to entity.RunMode, reason string) error {
	if err := m.stateMachine.Transition(to, reason); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Mode = to
	m.status.LastUpdatedAtUTC = time.Now().UTC()
	return nil
}

// CurrentState returns the current state from the state machine.
func (m *Manager) CurrentState() entity.RunMode {
	return m.stateMachine.Current()
}

// SubscribeStateChanges registers a listener for state transitions.
func (m *Manager) SubscribeStateChanges(ch chan<- StateTransition) uint64 {
	return m.stateMachine.Subscribe(ch)
}

// SetConnectivity updates the connectivity state.
func (m *Manager) SetConnectivity(state entity.ConnectivityState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Connectivity = state
	m.status.LastUpdatedAtUTC = time.Now().UTC()
}

// SetLicense updates the license state and banner.
func (m *Manager) SetLicense(state entity.LicenseState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.License = state
	if state == entity.LicenseStateRiskOnly {
		m.status.Banner = "license expired"
	}
	m.status.LastUpdatedAtUTC = time.Now().UTC()
}

// UpdateNTPDrift updates the NTP drift gate per instruction 8.3.
func (m *Manager) UpdateNTPDrift(drift time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.NTPDrift = drift
	if drift > 500*time.Millisecond {
		m.status.NewEntriesBlocked = true
	} else if drift < 200*time.Millisecond {
		m.status.NewEntriesBlocked = false
	}
	m.status.LastUpdatedAtUTC = time.Now().UTC()
}

// SetIntegrity updates the integrity status.
func (m *Manager) SetIntegrity(status entity.IntegrityState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Integrity = status
	m.status.LastUpdatedAtUTC = time.Now().UTC()
}

// EnableAnalyticsDegrade marks analytics degradation.
func (m *Manager) EnableAnalyticsDegrade() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.AnalyticsDegraded = true
	m.status.LastUpdatedAtUTC = time.Now().UTC()
}

// StartLive checks whether live trading may start per instruction 8.3 preflight.
func (m *Manager) StartLive(disclaimerAccepted bool) error {
	if !disclaimerAccepted {
		return ErrLiveTradingRequiresDisclaimer
	}
	return m.Transition(entity.RunModeLiveTrading, "live trading requested")
}

// StartPaper transitions into paper trading.
func (m *Manager) StartPaper() error {
	return m.Transition(entity.RunModePaperTrading, "paper trading requested")
}

// StopAll transitions back to idle.
func (m *Manager) StopAll() error {
	return m.Transition(entity.RunModeIdle, "stop all requested")
}

// ApplyMemoryLimit sets the Go soft memory limit.
func ApplyMemoryLimit(limit int64) int64 {
	return debug.SetMemoryLimit(limit)
}

// RunNTPWorker executes a periodic NTP gate callback.
func RunNTPWorker(ctx context.Context, interval time.Duration, probe func(context.Context) (time.Duration, error), onDrift func(time.Duration)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drift, err := probe(ctx)
			if err == nil {
				onDrift(drift)
			}
		}
	}
}
