package auditprovider

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/audit"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

// Repository adapts append-only audit logging to service-layer interfaces.
type Repository struct {
	logger  *audit.Logger
	version string
}

// New creates an audit provider repository.
func New(logger *audit.Logger, version string) *Repository {
	return &Repository{logger: logger, version: version}
}

// DisclaimerAccepted reports whether the operator accepted the live-trading disclaimer.
func (r *Repository) DisclaimerAccepted(context.Context) (bool, error) {
	return r.logger.HasAction(audit.ActionAcceptedFinancialRiskDisclaimer)
}

// AcceptDisclaimer records the disclaimer acceptance.
func (r *Repository) AcceptDisclaimer(_ context.Context, source string) error {
	return r.logger.Write(audit.DisclaimerRecord(source, entity.FinancialRiskDisclaimer, r.version))
}
