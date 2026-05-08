package vaultprovider

import (
	"context"
	"time"

	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/vault"
)

// Repository adapts vault storage to service-layer interfaces.
type Repository struct {
	store *vault.Store
	clock func() time.Time
}

// New creates a vault provider repository.
func New(store *vault.Store) *Repository {
	return &Repository{
		store: store,
		clock: func() time.Time { return time.Now().UTC() },
	}
}

// TrustPluginKey stores a trusted plugin public key after service-level validation.
func (r *Repository) TrustPluginKey(ctx context.Context, input serviceagent.TrustPluginKeyInput) error {
	return r.store.StoreTrustedPluginKey(ctx, vault.TrustedPluginKey{
		Fingerprint: input.Fingerprint,
		PublicKey:   input.PublicKey,
		AddedAtUTC:  r.clock(),
	})
}
