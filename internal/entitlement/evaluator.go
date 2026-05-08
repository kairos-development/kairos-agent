package entitlement

import (
	"time"

	"github.com/kairos-development/kairos-agent/internal/license"
)

type Evaluator struct {
	now func() time.Time
}

func NewEvaluator() *Evaluator {
	return &Evaluator{now: time.Now}
}

func (e *Evaluator) Evaluate(claims license.Claims, graceStartedAt *time.Time) Entitlement {
	now := e.now()
	state := license.EvaluateState(now, claims, graceStartedAt)

	edition := EditionCommunity
	if claims.Subject != "" {
		edition = parseEdition(claims.Subject)
	}

	policy := edition.Policy()

	ent := Entitlement{
		Edition:  edition,
		State:    string(state),
		Features: policy.AllowedFeatures,
		Limits:   Limits{MaxExchanges: policy.MaxExchanges, MaxStrategies: policy.MaxStrategies, MaxConnectors: policy.MaxConnectors},
	}

	if claims.Expiry > 0 {
		ent.ExpiresAt = time.Unix(claims.Expiry, 0).UTC()
	}

	if state == license.StateGrace && graceStartedAt != nil {
		ent.GraceUntil = graceStartedAt.UTC().Add(48 * time.Hour)
	}

	return ent
}

func parseEdition(sub string) Edition {
	switch sub {
	case "pro":
		return EditionPro
	case "team":
		return EditionTeam
	case "enterprise":
		return EditionEnterprise
	default:
		return EditionCommunity
	}
}

func (e *Evaluator) CanUseFeature(ent Entitlement, feature Feature) bool {
	if ent.State == string(license.StateRiskOnly) {
		return false
	}
	policy := Edition(ent.Edition).Policy()
	return policy.HasFeature(feature)
}

func (e *Evaluator) WithinLimits(ent Entitlement, kind string, current int) bool {
	policy := Edition(ent.Edition).Policy()
	switch kind {
	case "exchanges":
		if policy.Unlimited("exchanges") {
			return true
		}
		return current < policy.MaxExchanges
	case "strategies":
		if policy.Unlimited("strategies") {
			return true
		}
		return current < policy.MaxStrategies
	case "connectors":
		if policy.Unlimited("connectors") {
			return true
		}
		return current < policy.MaxConnectors
	}
	return false
}
