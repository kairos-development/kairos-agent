package runtimeprovider

import (
	"context"
	"fmt"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	providerconv "github.com/kairos-development/kairos-agent/internal/providers/conv"
	"github.com/kairos-development/kairos-agent/internal/runtime"
)

// Repository adapts runtime state management to service-layer interfaces.
type Repository struct {
	manager *runtime.Manager
	store   *SnapshotStore
}

// Option customizes runtime repository persistence behavior.
type Option func(*Repository)

// WithSnapshotStore enables crash-safe runtime status snapshots.
func WithSnapshotStore(store *SnapshotStore) Option {
	return func(r *Repository) {
		r.store = store
	}
}

// New creates a runtime provider repository.
func New(manager *runtime.Manager, opts ...Option) *Repository {
	repo := &Repository{manager: manager}
	for _, opt := range opts {
		if opt != nil {
			opt(repo)
		}
	}
	repo.restoreSnapshot(context.Background())
	return repo
}

// Status returns the domain runtime status.
func (r *Repository) Status(context.Context) (entity.RuntimeStatus, error) {
	return providerconv.RuntimeStatusToDomain(r.manager.Snapshot()), nil
}

// Transition attempts a state change through the runtime manager.
func (r *Repository) Transition(ctx context.Context, to entity.RunMode, reason string) error {
	if err := r.manager.Transition(to, reason); err != nil {
		return err
	}
	return r.persist(ctx)
}

// SetLicense updates the runtime license state.
func (r *Repository) SetLicense(ctx context.Context, state entity.LicenseState) error {
	r.manager.SetLicense(providerconv.LicenseToRuntime(state))
	return r.persist(ctx)
}

// SetConnectivity updates the runtime connectivity state.
func (r *Repository) SetConnectivity(ctx context.Context, state entity.ConnectivityState) error {
	r.manager.SetConnectivity(providerconv.ConnectivityToRuntime(state))
	return r.persist(ctx)
}

func (r *Repository) restoreSnapshot(ctx context.Context) {
	if r == nil || r.store == nil || r.manager == nil {
		return
	}

	status, found, err := r.store.Load(ctx)
	if err != nil {
		r.manager.RestoreSafetySnapshot(runtime.Status{
			Mode:              entity.RunModeHalted,
			Connectivity:      entity.ConnectivityStateNetworkWait,
			License:           entity.LicenseStateDemo,
			Integrity:         entity.IntegrityStateTrusted,
			NewEntriesBlocked: true,
			HaltReason:        fmt.Sprintf("runtime snapshot restore failed: %v", err),
		})
		_ = r.persist(context.Background())
		return
	}
	if !found {
		_ = r.persist(ctx)
		return
	}

	r.manager.RestoreSafetySnapshot(providerconv.RuntimeStatusFromDomain(status))
	_ = r.persist(ctx)
}

func (r *Repository) persist(ctx context.Context) error {
	if r == nil || r.store == nil {
		return nil
	}

	status := providerconv.RuntimeStatusToDomain(r.manager.Snapshot())
	if err := r.store.Save(ctx, status); err != nil {
		return fmt.Errorf("persist runtime snapshot: %w", err)
	}
	return nil
}
