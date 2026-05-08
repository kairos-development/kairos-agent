package entitlement

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/license"
	"github.com/stretchr/testify/assert"
)

func TestEditionPolicy_Community(t *testing.T) {
	p := EditionCommunity.Policy()
	assert.Equal(t, EditionCommunity, p.Edition)
	assert.Equal(t, 1, p.MaxExchanges)
	assert.Equal(t, 5, p.MaxStrategies)
	assert.Equal(t, 2, p.MaxConnectors)
	assert.False(t, p.HasFeature(FeatureCloudSync))
}

func TestEditionPolicy_Pro(t *testing.T) {
	p := EditionPro.Policy()
	assert.Equal(t, EditionPro, p.Edition)
	assert.Equal(t, 3, p.MaxExchanges)
	assert.True(t, p.HasFeature(FeatureCloudSync))
	assert.True(t, p.HasFeature(FeatureWebhookAlerts))
	assert.False(t, p.HasFeature(FeatureSSO))
}

func TestEditionPolicy_Team(t *testing.T) {
	p := EditionTeam.Policy()
	assert.Equal(t, EditionTeam, p.Edition)
	assert.Equal(t, 10, p.MaxExchanges)
	assert.True(t, p.HasFeature(FeatureStrategyShare))
	assert.True(t, p.HasFeature(FeaturePrioritySupport))
	assert.True(t, p.HasFeature(FeatureAuditLog))
}

func TestEditionPolicy_Enterprise(t *testing.T) {
	p := EditionEnterprise.Policy()
	assert.Equal(t, EditionEnterprise, p.Edition)
	assert.True(t, p.Unlimited("exchanges"))
	assert.True(t, p.Unlimited("strategies"))
	assert.True(t, p.Unlimited("connectors"))
	assert.True(t, p.HasFeature(FeatureSSO))
	assert.True(t, p.HasFeature(FeatureCustomPlugins))
}

func TestPolicy_HasFeature(t *testing.T) {
	p := Policy{AllowedFeatures: []Feature{FeatureCloudSync, FeatureWebhookAlerts}}
	assert.True(t, p.HasFeature(FeatureCloudSync))
	assert.True(t, p.HasFeature(FeatureWebhookAlerts))
	assert.False(t, p.HasFeature(FeatureSSO))
}

func TestPolicy_Unlimited(t *testing.T) {
	tests := []struct {
		name string
		p    Policy
		kind string
		want bool
	}{
		{"unlimited exchanges", Policy{MaxExchanges: -1}, "exchanges", true},
		{"limited exchanges", Policy{MaxExchanges: 3}, "exchanges", false},
		{"unlimited strategies", Policy{MaxStrategies: -1}, "strategies", true},
		{"unlimited connectors", Policy{MaxConnectors: -1}, "connectors", true},
		{"unknown kind", Policy{}, "unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.p.Unlimited(tt.kind))
		})
	}
}

func TestEvaluator_Evaluate_Demo(t *testing.T) {
	ev := NewEvaluator()
	claims := license.Claims{Subject: "demo", HWID: "hwid", Expiry: 0}
	ent := ev.Evaluate(claims, nil)

	assert.Equal(t, EditionCommunity, ent.Edition)
	assert.Equal(t, "grace", ent.State)
}

func TestEvaluator_Evaluate_Licensed(t *testing.T) {
	ev := NewEvaluator()
	expiry := time.Now().Add(24 * time.Hour).Unix()
	claims := license.Claims{Subject: "pro", HWID: "hwid", Expiry: expiry}
	ent := ev.Evaluate(claims, nil)

	assert.Equal(t, EditionPro, ent.Edition)
	assert.Equal(t, "licensed", ent.State)
	assert.True(t, ent.ExpiresAt.After(time.Now()))
}

func TestEvaluator_Evaluate_ExpiredToRiskOnly(t *testing.T) {
	ev := &Evaluator{now: func() time.Time { return time.Now() }}
	expiry := time.Now().Add(-49 * time.Hour).Unix()
	claims := license.Claims{Subject: "pro", HWID: "hwid", Expiry: expiry}
	graceStart := time.Now().Add(-49 * time.Hour)
	ent := ev.Evaluate(claims, &graceStart)

	assert.Equal(t, "risk_only", ent.State)
}

func TestEvaluator_CanUseFeature(t *testing.T) {
	ev := NewEvaluator()
	ent := Entitlement{
		Edition:  EditionPro,
		State:    "licensed",
		Features: []Feature{FeatureCloudSync, FeatureWebhookAlerts},
	}

	assert.True(t, ev.CanUseFeature(ent, FeatureCloudSync))
	assert.True(t, ev.CanUseFeature(ent, FeatureWebhookAlerts))
	assert.False(t, ev.CanUseFeature(ent, FeatureSSO))
}

func TestEvaluator_CanUseFeature_RiskOnly(t *testing.T) {
	ev := NewEvaluator()
	ent := Entitlement{
		Edition:  EditionEnterprise,
		State:    "risk_only",
		Features: []Feature{FeatureSSO, FeatureCloudSync},
	}

	assert.False(t, ev.CanUseFeature(ent, FeatureSSO))
	assert.False(t, ev.CanUseFeature(ent, FeatureCloudSync))
}

func TestEvaluator_WithinLimits(t *testing.T) {
	ev := NewEvaluator()
	ent := Entitlement{
		Edition: EditionPro,
		Limits:  Limits{MaxExchanges: 3, MaxStrategies: 50, MaxConnectors: 10},
	}

	assert.True(t, ev.WithinLimits(ent, "exchanges", 2))
	assert.False(t, ev.WithinLimits(ent, "exchanges", 3))
	assert.True(t, ev.WithinLimits(ent, "strategies", 49))
	assert.True(t, ev.WithinLimits(ent, "connectors", 5))
}

func TestEvaluator_WithinLimits_Unlimited(t *testing.T) {
	ev := NewEvaluator()
	ent := Entitlement{
		Edition: EditionEnterprise,
		Limits:  Limits{MaxExchanges: -1, MaxStrategies: -1, MaxConnectors: -1},
	}

	assert.True(t, ev.WithinLimits(ent, "exchanges", 1000))
	assert.True(t, ev.WithinLimits(ent, "strategies", 5000))
}

func TestParseEdition(t *testing.T) {
	tests := []struct {
		sub  string
		want Edition
	}{
		{"pro", EditionPro},
		{"team", EditionTeam},
		{"enterprise", EditionEnterprise},
		{"demo", EditionCommunity},
		{"", EditionCommunity},
		{"unknown", EditionCommunity},
	}
	for _, tt := range tests {
		t.Run(tt.sub, func(t *testing.T) {
			ev := NewEvaluator()
			claims := license.Claims{Subject: tt.sub}
			ent := ev.Evaluate(claims, nil)
			assert.Equal(t, tt.want, ent.Edition)
		})
	}
}
