package runtimeprovider

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	providerconv "github.com/kairos-development/kairos-agent/internal/providers/conv"
	"github.com/kairos-development/kairos-agent/internal/runtime"
)

// Repository adapts runtime state management to service-layer interfaces.
type Repository struct {
	manager *runtime.Manager
}

// New creates a runtime provider repository.
func New(manager *runtime.Manager) *Repository {
	return &Repository{manager: manager}
}

// Status returns the domain runtime status.
func (r *Repository) Status(context.Context) (entity.RuntimeStatus, error) {
	return providerconv.RuntimeStatusToDomain(r.manager.Snapshot()), nil
}

// Transition attempts a state change through the runtime manager.
func (r *Repository) Transition(_ context.Context, to entity.RunMode, reason string) error {
	return r.manager.Transition(to, reason)
}

// SetLicense updates the runtime license state.
func (r *Repository) SetLicense(_ context.Context, state entity.LicenseState) error {
	r.manager.SetLicense(providerconv.LicenseToRuntime(state))
	return nil
}

// SetConnectivity updates the runtime connectivity state.
func (r *Repository) SetConnectivity(_ context.Context, state entity.ConnectivityState) error {
	r.manager.SetConnectivity(providerconv.ConnectivityToRuntime(state))
	return nil
}
