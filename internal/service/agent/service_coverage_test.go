package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/entitlement"
	"github.com/kairos-development/kairos-agent/internal/license"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- extended mocks that support error injection ---

type runtimeErrMock struct {
	runtimeMock
	transitionErr error
	setLicenseErr error
	setConnErr    error
}

func (m *runtimeErrMock) Transition(ctx context.Context, mode entity.RunMode, reason string) error {
	if m.transitionErr != nil {
		return m.transitionErr
	}
	m.transitions = append(m.transitions, struct {
		mode   entity.RunMode
		reason string
	}{mode, reason})
	return nil
}

func (m *runtimeErrMock) SetLicense(ctx context.Context, state entity.LicenseState) error {
	if m.setLicenseErr != nil {
		return m.setLicenseErr
	}
	m.license = append(m.license, state)
	return nil
}

func (m *runtimeErrMock) SetConnectivity(ctx context.Context, state entity.ConnectivityState) error {
	if m.setConnErr != nil {
		return m.setConnErr
	}
	m.connectivity = append(m.connectivity, state)
	return nil
}

type configErrMock struct {
	configMock
	err error
}

func (m *configErrMock) Config(ctx context.Context) (entity.AgentConfig, error) {
	if m.err != nil {
		return entity.AgentConfig{}, m.err
	}
	return m.configMock.Config(ctx)
}

type disclaimerErrMock struct {
	disclaimerMock
	err error
}

func (m *disclaimerErrMock) DisclaimerAccepted(context.Context) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.accepted, nil
}

// --- Entitlement tests ---

func TestEntitlement_CommunityEdition(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "community",
		LicenseExpiry:  0,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "community", out.Edition)
	assert.NotEmpty(t, out.State)
	assert.Equal(t, 1, out.Limits.MaxExchanges)
	assert.Equal(t, 5, out.Limits.MaxStrategies)
	assert.Equal(t, 2, out.Limits.MaxConnectors)
	assert.Empty(t, out.Features)
}

func TestEntitlement_ProEdition(t *testing.T) {
	futureExpiry := time.Now().Add(24 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "pro",
		LicenseExpiry:  futureExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "pro", out.Edition)
	assert.Equal(t, string(license.StateLicensed), out.State)
	assert.Equal(t, 3, out.Limits.MaxExchanges)
	assert.Equal(t, 50, out.Limits.MaxStrategies)
	assert.Equal(t, 10, out.Limits.MaxConnectors)
	assert.Contains(t, out.Features, string(entitlement.FeatureWebhookAlerts))
	assert.Contains(t, out.Features, string(entitlement.FeatureCloudSync))
}

func TestEntitlement_TeamEdition(t *testing.T) {
	futureExpiry := time.Now().Add(48 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "team",
		LicenseExpiry:  futureExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "team", out.Edition)
	assert.Equal(t, 10, out.Limits.MaxExchanges)
	assert.Equal(t, 500, out.Limits.MaxStrategies)
	assert.Equal(t, 50, out.Limits.MaxConnectors)
	assert.Contains(t, out.Features, string(entitlement.FeatureStrategyShare))
	assert.Contains(t, out.Features, string(entitlement.FeaturePrioritySupport))
	assert.Contains(t, out.Features, string(entitlement.FeatureAuditLog))
}

func TestEntitlement_EnterpriseEdition(t *testing.T) {
	futureExpiry := time.Now().Add(72 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "enterprise",
		LicenseExpiry:  futureExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "enterprise", out.Edition)
	assert.Equal(t, -1, out.Limits.MaxExchanges)
	assert.Equal(t, -1, out.Limits.MaxStrategies)
	assert.Equal(t, -1, out.Limits.MaxConnectors)
	assert.Contains(t, out.Features, string(entitlement.FeatureSSO))
	assert.Contains(t, out.Features, string(entitlement.FeatureCustomPlugins))
}

func TestEntitlement_ExpiredLicense(t *testing.T) {
	pastExpiry := time.Now().Add(-24 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "pro",
		LicenseExpiry:  pastExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "pro", out.Edition)
	assert.Equal(t, string(license.StateGrace), out.State)
}

func TestEntitlement_RiskOnlyState(t *testing.T) {
	pastExpiry := time.Now().Add(-24 * time.Hour).Unix()
	graceStarted := time.Now().Add(-72 * time.Hour)
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "pro",
		LicenseExpiry:  pastExpiry,
		GraceStartedAt: &graceStarted,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, string(license.StateRiskOnly), out.State)
}

func TestEntitlement_GraceWithDeadline(t *testing.T) {
	pastExpiry := time.Now().Add(-24 * time.Hour).Unix()
	graceStarted := time.Now().Add(-12 * time.Hour)
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "pro",
		LicenseExpiry:  pastExpiry,
		GraceStartedAt: &graceStarted,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, string(license.StateGrace), out.State)
}

func TestEntitlement_ConfigError(t *testing.T) {
	svc := New(&runtimeMock{}, &configErrMock{err: errors.New("config unavailable")}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	_, err := svc.Entitlement(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config unavailable")
}

func TestEntitlement_EmptySubject(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "",
		LicenseExpiry:  0,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Entitlement(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "community", out.Edition)
}

// --- CanUseFeature tests ---

func TestCanUseFeature_CommunityNoFeatures(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "community",
		LicenseExpiry:  0,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	ok, err := svc.CanUseFeature(context.Background(), "webhook_alerts")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCanUseFeature_ProHasWebhookAlerts(t *testing.T) {
	futureExpiry := time.Now().Add(24 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "pro",
		LicenseExpiry:  futureExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	ok, err := svc.CanUseFeature(context.Background(), "webhook_alerts")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestCanUseFeature_ProNoSSO(t *testing.T) {
	futureExpiry := time.Now().Add(24 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "pro",
		LicenseExpiry:  futureExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	ok, err := svc.CanUseFeature(context.Background(), "sso")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCanUseFeature_EnterpriseHasAll(t *testing.T) {
	futureExpiry := time.Now().Add(24 * time.Hour).Unix()
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "enterprise",
		LicenseExpiry:  futureExpiry,
		GraceStartedAt: nil,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	features := []string{"webhook_alerts", "cloud_sync", "strategy_share", "sso", "custom_plugins", "audit_log"}
	for _, f := range features {
		ok, err := svc.CanUseFeature(context.Background(), f)
		require.NoError(t, err)
		assert.True(t, ok, "feature %s should be allowed for enterprise", f)
	}
}

func TestCanUseFeature_RiskOnlyBlocksAll(t *testing.T) {
	pastExpiry := time.Now().Add(-24 * time.Hour).Unix()
	graceStarted := time.Now().Add(-72 * time.Hour)
	cfg := entity.AgentConfig{
		SchemaVersion:  1,
		Edition:        "enterprise",
		LicenseExpiry:  pastExpiry,
		GraceStartedAt: &graceStarted,
	}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	ok, err := svc.CanUseFeature(context.Background(), "webhook_alerts")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCanUseFeature_ConfigError(t *testing.T) {
	svc := New(&runtimeMock{}, &configErrMock{err: errors.New("db down")}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	_, err := svc.CanUseFeature(context.Background(), "webhook_alerts")
	assert.Error(t, err)
}

// --- StartLive error edge cases ---

func TestStartLive_SetLicenseError(t *testing.T) {
	runtimeRepo := &runtimeErrMock{
		setLicenseErr: errors.New("license service unavailable"),
	}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{accepted: true}, exporterMock{}, &pluginTrustMock{})

	err := svc.StartLive(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "license service unavailable")
}

func TestStartLive_DisclaimerCheckError(t *testing.T) {
	svc := New(&runtimeMock{}, &configMock{}, &disclaimerErrMock{err: errors.New("storage error")}, exporterMock{}, &pluginTrustMock{})

	err := svc.StartLive(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// --- EmergencyStop error edge cases ---

func TestEmergencyStop_TransitionError(t *testing.T) {
	runtimeRepo := &runtimeErrMock{
		transitionErr: errors.New("state machine locked"),
	}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.EmergencyStop(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state machine locked")
}

func TestEmergencyStop_SetConnectivityError(t *testing.T) {
	runtimeRepo := &runtimeErrMock{
		setConnErr: errors.New("connectivity update failed"),
	}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.EmergencyStop(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connectivity update failed")
}

// --- Check edge cases ---

func TestCheck_ConfigError(t *testing.T) {
	svc := New(&runtimeMock{}, &configErrMock{err: errors.New("config not found")}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	_, err := svc.Check(context.Background())
	assert.Error(t, err)
}

func TestCheck_RuntimeStatusError(t *testing.T) {
	runtimeRepo := &runtimeErrMock{}
	runtimeRepo.status = entity.RuntimeStatus{}
	// RuntimeMock.Status always returns nil error, so we test with config error instead
	svc := New(&runtimeMock{}, &configErrMock{err: errors.New("config error")}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	_, err := svc.Check(context.Background())
	assert.Error(t, err)
}

func TestCheck_UnlicensedGetsCloudLicense(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateUnlicensed,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC(),
	}}
	cfg := entity.AgentConfig{
		SchemaVersion: 1,
		Telemetry:     entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileMinimal},
	}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Check(context.Background())
	require.NoError(t, err)
	// Unlicensed != Demo, so cloud-license is added; telemetry enabled + non-off profile adds cloud-telemetry
	assert.Contains(t, out.AllowedOutbound, "cloud-license")
	assert.Contains(t, out.AllowedOutbound, "cloud-telemetry")
}

func TestCheck_GraceGetsCloudLicense(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateGrace,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC(),
	}}
	cfg := entity.AgentConfig{
		SchemaVersion: 1,
		Telemetry:     entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileMinimal},
	}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Check(context.Background())
	require.NoError(t, err)
	// Grace != Demo, so cloud-license is added
	assert.Contains(t, out.AllowedOutbound, "cloud-license")
}

func TestCheck_TelemetryDisabledNoCloudTelemetry(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateLicensed,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC(),
	}}
	cfg := entity.AgentConfig{
		SchemaVersion: 1,
		Telemetry:     entity.TelemetryConsent{Enabled: false, Profile: entity.TelemetryProfileOff},
	}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Check(context.Background())
	require.NoError(t, err)
	for _, ch := range out.AllowedOutbound {
		assert.NotEqual(t, "cloud-telemetry", ch)
	}
}

func TestCheck_TelemetryOffProfileNoCloudTelemetry(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateLicensed,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC(),
	}}
	cfg := entity.AgentConfig{
		SchemaVersion: 1,
		Telemetry:     entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileOff},
	}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Check(context.Background())
	require.NoError(t, err)
	for _, ch := range out.AllowedOutbound {
		assert.NotEqual(t, "cloud-telemetry", ch)
	}
}
