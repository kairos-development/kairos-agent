package tradinggate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runtimeMock struct {
	status entity.RuntimeStatus
	err    error
}

func (m *runtimeMock) Status(context.Context) (entity.RuntimeStatus, error) {
	return m.status, m.err
}

func TestGate_CheckNewEntry_AllowsPaperTrading(t *testing.T) {
	gate := New(&runtimeMock{status: entity.RuntimeStatus{
		Mode:         entity.RunModePaperTrading,
		Connectivity: entity.ConnectivityStateConnected,
		Integrity:    entity.IntegrityStateTrusted,
		License:      entity.LicenseStateDemo,
		NTPDrift:     50 * time.Millisecond,
	}}, DefaultPolicy())

	require.NoError(t, gate.CheckNewEntry(context.Background()))
}

func TestGate_CheckNewEntry_AllowsLicensedLiveTrading(t *testing.T) {
	gate := New(&runtimeMock{status: entity.RuntimeStatus{
		Mode:         entity.RunModeLiveTrading,
		Connectivity: entity.ConnectivityStateConnected,
		Integrity:    entity.IntegrityStateTrusted,
		License:      entity.LicenseStateLicensed,
		NTPDrift:     50 * time.Millisecond,
	}}, DefaultPolicy())

	require.NoError(t, gate.CheckNewEntry(context.Background()))
}

func TestGate_CheckNewEntry_BlocksUnsafeStates(t *testing.T) {
	tests := []struct {
		name   string
		status entity.RuntimeStatus
	}{
		{
			name: "halted",
			status: entity.RuntimeStatus{
				Mode:         entity.RunModeHalted,
				Connectivity: entity.ConnectivityStateConnected,
				Integrity:    entity.IntegrityStateTrusted,
				License:      entity.LicenseStateLicensed,
			},
		},
		{
			name: "reconnecting",
			status: entity.RuntimeStatus{
				Mode:         entity.RunModeLiveTrading,
				Connectivity: entity.ConnectivityStateReconnecting,
				Integrity:    entity.IntegrityStateTrusted,
				License:      entity.LicenseStateLicensed,
			},
		},
		{
			name: "explicit block",
			status: entity.RuntimeStatus{
				Mode:              entity.RunModeLiveTrading,
				Connectivity:      entity.ConnectivityStateConnected,
				Integrity:         entity.IntegrityStateTrusted,
				License:           entity.LicenseStateLicensed,
				NewEntriesBlocked: true,
			},
		},
		{
			name: "degraded integrity",
			status: entity.RuntimeStatus{
				Mode:         entity.RunModeLiveTrading,
				Connectivity: entity.ConnectivityStateConnected,
				Integrity:    entity.IntegrityStateDegraded,
				License:      entity.LicenseStateLicensed,
			},
		},
		{
			name: "stale ntp",
			status: entity.RuntimeStatus{
				Mode:         entity.RunModeLiveTrading,
				Connectivity: entity.ConnectivityStateConnected,
				Integrity:    entity.IntegrityStateTrusted,
				License:      entity.LicenseStateLicensed,
				NTPDrift:     time.Second,
			},
		},
		{
			name: "unlicensed live",
			status: entity.RuntimeStatus{
				Mode:         entity.RunModeLiveTrading,
				Connectivity: entity.ConnectivityStateConnected,
				Integrity:    entity.IntegrityStateTrusted,
				License:      entity.LicenseStateUnlicensed,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gate := New(&runtimeMock{status: tt.status}, DefaultPolicy())

			err := gate.CheckNewEntry(context.Background())
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBlocked)
		})
	}
}

func TestGate_CheckNewEntry_StatusUnavailable(t *testing.T) {
	gate := New(&runtimeMock{err: assert.AnError}, DefaultPolicy())

	err := gate.CheckNewEntry(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStatusUnavailable)
}

func TestGate_CheckNewEntry_NilGateAllows(t *testing.T) {
	var gate *Gate

	require.NoError(t, gate.CheckNewEntry(context.Background()))
}

func TestBlockError_UnwrapsBlocked(t *testing.T) {
	err := newBlockError("test", entity.RuntimeStatus{})

	assert.True(t, errors.Is(err, ErrBlocked))
}
