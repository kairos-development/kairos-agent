package agent

import (
	"context"
	"fmt"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/entitlement"
	"github.com/kairos-development/kairos-agent/internal/license"
)

// ErrFinancialRiskDisclaimerRequired reports that live trading needs an accepted disclaimer.
var ErrFinancialRiskDisclaimerRequired = fmt.Errorf("live trading requires accepted financial risk disclaimer")

type service struct {
	runtime     runtimeRepository
	config      configRepository
	disclaimer  disclaimerRepository
	exporter    exportRepository
	plugins     pluginTrustRepository
	entitlement *entitlement.Evaluator
}

// New creates the agent application service.
func New(runtime runtimeRepository, config configRepository, disclaimer disclaimerRepository, exporter exportRepository, plugins pluginTrustRepository) Service {
	return &service{
		runtime:     runtime,
		config:      config,
		disclaimer:  disclaimer,
		exporter:    exporter,
		plugins:     plugins,
		entitlement: entitlement.NewEvaluator(),
	}
}

// AcceptDisclaimer records the operator disclaimer acceptance.
func (s *service) AcceptDisclaimer(ctx context.Context, source string) error {
	return s.disclaimer.AcceptDisclaimer(ctx, source)
}

// Check builds the service-level health report.
func (s *service) Check(ctx context.Context) (CheckOutput, error) {
	cfg, err := s.config.Config(ctx)
	if err != nil {
		return CheckOutput{}, err
	}
	status, err := s.runtime.Status(ctx)
	if err != nil {
		return CheckOutput{}, err
	}
	allowedOutbound := []string{"exchange-public", "exchange-private"}
	if status.License != entity.LicenseStateDemo {
		allowedOutbound = append(allowedOutbound, "cloud-license")
		if cfg.Telemetry.Enabled && cfg.Telemetry.Profile != entity.TelemetryProfileOff {
			allowedOutbound = append(allowedOutbound, "cloud-telemetry")
		}
	}
	return CheckOutput{
		SchemaVersion:   cfg.SchemaVersion,
		Telemetry:       cfg.Telemetry,
		AllowedOutbound: allowedOutbound,
		Status:          status,
	}, nil
}

// Status returns the domain runtime status.
func (s *service) Status(ctx context.Context) (entity.RuntimeStatus, error) {
	return s.runtime.Status(ctx)
}

// Config returns the domain agent configuration.
func (s *service) Config(ctx context.Context) (entity.AgentConfig, error) {
	return s.config.Config(ctx)
}

// StartScan switches the runtime into scanning mode.
func (s *service) StartScan(ctx context.Context) error {
	return s.runtime.Transition(ctx, entity.RunModeScanning, "scanning requested")
}

// StartPaper switches the runtime into paper trading mode.
func (s *service) StartPaper(ctx context.Context) error {
	return s.runtime.Transition(ctx, entity.RunModePaperTrading, "paper trading requested")
}

// Entitlement returns the current entitlement information.
func (s *service) Entitlement(ctx context.Context) (EntitlementOutput, error) {
	cfg, err := s.config.Config(ctx)
	if err != nil {
		return EntitlementOutput{}, err
	}
	claims := license.Claims{Subject: cfg.Edition, Expiry: cfg.LicenseExpiry}
	ent := s.entitlement.Evaluate(claims, cfg.GraceStartedAt)
	features := make([]string, len(ent.Features))
	for i, f := range ent.Features {
		features[i] = string(f)
	}
	return EntitlementOutput{
		Edition:  string(ent.Edition),
		State:    ent.State,
		Limits:   entity.Limits{MaxExchanges: ent.Limits.MaxExchanges, MaxStrategies: ent.Limits.MaxStrategies, MaxConnectors: ent.Limits.MaxConnectors},
		Features: features,
	}, nil
}

// CanUseFeature checks if a feature is allowed under current entitlement.
func (s *service) CanUseFeature(ctx context.Context, feature string) (bool, error) {
	cfg, err := s.config.Config(ctx)
	if err != nil {
		return false, err
	}
	claims := license.Claims{Subject: cfg.Edition, Expiry: cfg.LicenseExpiry}
	ent := s.entitlement.Evaluate(claims, cfg.GraceStartedAt)
	return s.entitlement.CanUseFeature(ent, entitlement.Feature(feature)), nil
}

// StartLive switches the runtime into live trading mode if disclaimers are accepted.
func (s *service) StartLive(ctx context.Context) error {
	accepted, err := s.disclaimer.DisclaimerAccepted(ctx)
	if err != nil {
		return err
	}
	if !accepted {
		return ErrFinancialRiskDisclaimerRequired
	}
	if err := s.runtime.SetLicense(ctx, entity.LicenseStateLicensed); err != nil {
		return err
	}
	return s.runtime.Transition(ctx, entity.RunModeLiveTrading, "live trading requested")
}

// QueueBacktest switches the runtime into backtest mode.
func (s *service) QueueBacktest(ctx context.Context) error {
	return s.runtime.Transition(ctx, entity.RunModeBacktesting, "backtest requested")
}

// StopAll returns the runtime to idle.
func (s *service) StopAll(ctx context.Context) error {
	return s.runtime.Transition(ctx, entity.RunModeIdle, "stop all requested")
}

// EmergencyStop forces the runtime into halted network waiting state.
func (s *service) EmergencyStop(ctx context.Context) error {
	if err := s.runtime.Transition(ctx, entity.RunModeHalted, "emergency stop"); err != nil {
		return err
	}
	return s.runtime.SetConnectivity(ctx, entity.ConnectivityStateNetworkWait)
}

// ExportCSV requests a CSV export from the data layer.
func (s *service) ExportCSV(ctx context.Context, input ExportCSVInput) (ExportCSVOutput, error) {
	return s.exporter.ExportCSV(ctx, input)
}

// TrustPluginKey stores a trusted plugin signing key if it is allowed by config.
func (s *service) TrustPluginKey(ctx context.Context, input TrustPluginKeyInput) error {
	cfg, err := s.config.Config(ctx)
	if err != nil {
		return err
	}
	allowed := false
	for _, fingerprint := range cfg.TrustedPluginFingerprints {
		if fingerprint == input.Fingerprint {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("fingerprint %q is not listed in config trusted_plugin_keys", input.Fingerprint)
	}
	return s.plugins.TrustPluginKey(ctx, input)
}

// SetConfig sets a complete configuration.
func (s *service) SetConfig(ctx context.Context, input SetConfigInput) error {
	// Validate configuration before setting
	if input.Config.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema version: %d", input.Config.SchemaVersion)
	}
	return s.config.SetConfig(ctx, input.Config)
}

// UpdateConfig updates specific configuration fields.
func (s *service) UpdateConfig(ctx context.Context, input UpdateConfigInput) error {
	updates := map[string]interface{}{
		input.Key: input.Value,
	}
	return s.config.UpdateConfig(ctx, updates)
}
