// Package tradinggate centralizes runtime safety checks for new trade entries.
package tradinggate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

var (
	// ErrBlocked reports that new trade entries are currently blocked.
	ErrBlocked = errors.New("trading gate blocked new entries")

	// ErrStatusUnavailable reports that runtime status could not be read.
	ErrStatusUnavailable = errors.New("runtime status unavailable")
)

// RuntimeReader provides runtime state to the trading gate.
type RuntimeReader interface {
	// Status returns the current domain runtime status.
	Status(context.Context) (entity.RuntimeStatus, error)
}

// Policy configures runtime checks for new trade entries.
type Policy struct {
	AllowedModes       map[entity.RunMode]struct{}
	MaxNTPDrift        time.Duration
	RequireConnected   bool
	RequireTrusted     bool
	RequireLiveLicense bool
}

// DefaultPolicy returns the production safety policy for new trade entries.
func DefaultPolicy() Policy {
	return Policy{
		AllowedModes: map[entity.RunMode]struct{}{
			entity.RunModeLiveTrading:  {},
			entity.RunModePaperTrading: {},
		},
		MaxNTPDrift:        500 * time.Millisecond,
		RequireConnected:   true,
		RequireTrusted:     true,
		RequireLiveLicense: true,
	}
}

// Gate validates whether new trade entries may be created or submitted.
type Gate struct {
	runtime RuntimeReader
	policy  Policy
}

// New creates a runtime-backed trading gate.
func New(runtime RuntimeReader, policy Policy) *Gate {
	if policy.AllowedModes == nil {
		policy = DefaultPolicy()
	}
	return &Gate{runtime: runtime, policy: policy}
}

// CheckNewEntry returns nil only when a new trade entry is safe to create or submit.
func (g *Gate) CheckNewEntry(ctx context.Context) error {
	if g == nil || g.runtime == nil {
		return nil
	}

	status, err := g.runtime.Status(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStatusUnavailable, err)
	}

	if _, ok := g.policy.AllowedModes[status.Mode]; !ok {
		return newBlockError("runtime mode does not allow new entries", status)
	}

	if status.NewEntriesBlocked {
		return newBlockError("runtime explicitly blocked new entries", status)
	}

	if g.policy.RequireConnected && status.Connectivity != entity.ConnectivityStateConnected {
		return newBlockError("exchange connectivity is not healthy", status)
	}

	if g.policy.RequireTrusted && status.Integrity != entity.IntegrityStateTrusted {
		return newBlockError("runtime integrity is degraded", status)
	}

	if g.policy.MaxNTPDrift > 0 && status.NTPDrift > g.policy.MaxNTPDrift {
		return newBlockError("NTP drift exceeds allowed threshold", status)
	}

	if g.policy.RequireLiveLicense && status.Mode == entity.RunModeLiveTrading && !isLiveLicenseAllowed(status.License) {
		return newBlockError("live trading license is not valid for new entries", status)
	}

	return nil
}

func isLiveLicenseAllowed(state entity.LicenseState) bool {
	return state == entity.LicenseStateLicensed || state == entity.LicenseStateGrace
}

func newBlockError(reason string, status entity.RuntimeStatus) *BlockError {
	return &BlockError{Reason: reason, Status: status}
}

// BlockError describes why the trading gate rejected a new entry.
type BlockError struct {
	Reason string
	Status entity.RuntimeStatus
}

// Error returns the human-readable rejection reason.
func (e *BlockError) Error() string {
	return fmt.Sprintf("%s: %s", ErrBlocked, e.Reason)
}

// Unwrap returns the sentinel blocked error.
func (e *BlockError) Unwrap() error {
	return ErrBlocked
}
