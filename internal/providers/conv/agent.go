package conv

import (
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/runtime"
)

// ConfigToDomain converts the config-layer model into a domain config.
func ConfigToDomain(cfg *config.Config) entity.AgentConfig {
	return entity.AgentConfig{
		SchemaVersion:             cfg.SchemaVersion,
		ExportTimezone:            cfg.ExportTimezone,
		TrustedPluginFingerprints: append([]string(nil), cfg.TrustedPluginKeys...),
		Telemetry: entity.TelemetryConsent{
			Enabled:        cfg.Telemetry.Enabled,
			Profile:        entity.TelemetryProfile(cfg.Telemetry.Profile),
			ConsentVersion: cfg.Telemetry.ConsentVersion,
			ConsentSource:  cfg.Telemetry.ConsentSource,
			ConsentedAtUTC: cfg.Telemetry.ConsentedAtUTC,
			InstallID:      cfg.Telemetry.AnonymousInstallID,
		},
		JournalMaxSizeBytes:       cfg.JournalMaxSizeBytes,
		MemoryLimitBytes:          cfg.MemoryLimitBytes,
		MemoryDegradeThresholdPct: cfg.MemoryDegradeThresholdPct,
		Paths: entity.StatePaths{
			ConfigPath:  cfg.Paths.ConfigPath,
			TradesDB:    cfg.Paths.TradesDB,
			VaultDB:     cfg.Paths.VaultDB,
			JournalPath: cfg.Paths.JournalPath,
			AuditPath:   cfg.Paths.AuditPath,
		},
		Risk: entity.RiskConfig{MaxPosition: cfg.Risk.MaxPosition},
		Exchange: entity.ExchangeConfig{
			APIKeyID:      cfg.Exchange.APIKeyID,
			DefaultSymbol: cfg.Exchange.DefaultSymbol,
		},
		Edition:        cfg.Edition,
		LicenseExpiry:  cfg.LicenseExpiry,
		GraceStartedAt: cfg.GraceStartedAt,
	}
}

// ConfigFromDomain converts a domain config into the config-layer model.
func ConfigFromDomain(domainCfg entity.AgentConfig) *config.Config {
	return &config.Config{
		SchemaVersion:     domainCfg.SchemaVersion,
		ExportTimezone:    domainCfg.ExportTimezone,
		TrustedPluginKeys: append([]string(nil), domainCfg.TrustedPluginFingerprints...),
		Telemetry: config.TelemetryConsent{
			Enabled:            domainCfg.Telemetry.Enabled,
			Profile:            config.TelemetryProfile(domainCfg.Telemetry.Profile),
			ConsentVersion:     domainCfg.Telemetry.ConsentVersion,
			ConsentSource:      domainCfg.Telemetry.ConsentSource,
			ConsentedAtUTC:     domainCfg.Telemetry.ConsentedAtUTC,
			AnonymousInstallID: domainCfg.Telemetry.InstallID,
		},
		JournalMaxSizeBytes:       domainCfg.JournalMaxSizeBytes,
		MemoryLimitBytes:          domainCfg.MemoryLimitBytes,
		MemoryDegradeThresholdPct: domainCfg.MemoryDegradeThresholdPct,
		Paths: config.PathsConfig{
			ConfigPath:  domainCfg.Paths.ConfigPath,
			TradesDB:    domainCfg.Paths.TradesDB,
			VaultDB:     domainCfg.Paths.VaultDB,
			JournalPath: domainCfg.Paths.JournalPath,
			AuditPath:   domainCfg.Paths.AuditPath,
		},
		Risk: config.RiskConfig{MaxPosition: domainCfg.Risk.MaxPosition},
		Exchange: config.ExchangeConfig{
			APIKeyID:      domainCfg.Exchange.APIKeyID,
			DefaultSymbol: domainCfg.Exchange.DefaultSymbol,
		},
		Edition:        domainCfg.Edition,
		LicenseExpiry:  domainCfg.LicenseExpiry,
		GraceStartedAt: domainCfg.GraceStartedAt,
	}
}

// RuntimeStatusToDomain converts the runtime-layer model into a domain status.
func RuntimeStatusToDomain(status runtime.Status) entity.RuntimeStatus {
	var haltedAt *time.Time
	if status.HaltedAtUTC != nil {
		copied := *status.HaltedAtUTC
		haltedAt = &copied
	}
	return entity.RuntimeStatus{
		Mode:              runModeFromRuntime(status.Mode),
		Connectivity:      connectivityFromRuntime(status.Connectivity),
		License:           licenseFromRuntime(status.License),
		Integrity:         integrityFromRuntime(status.Integrity),
		NTPDrift:          status.NTPDrift,
		NewEntriesBlocked: status.NewEntriesBlocked,
		HaltReason:        status.HaltReason,
		HaltedAtUTC:       haltedAt,
		Banner:            status.Banner,
		AnalyticsDegraded: status.AnalyticsDegraded,
		LastUpdatedAtUTC:  status.LastUpdatedAtUTC,
	}
}

// RuntimeStatusFromDomain converts a domain runtime status into the runtime-layer model.
func RuntimeStatusFromDomain(status entity.RuntimeStatus) runtime.Status {
	var haltedAt *time.Time
	if status.HaltedAtUTC != nil {
		copied := *status.HaltedAtUTC
		haltedAt = &copied
	}
	return runtime.Status{
		Mode:              RunModeToRuntime(status.Mode),
		Connectivity:      ConnectivityToRuntime(status.Connectivity),
		License:           LicenseToRuntime(status.License),
		Integrity:         entity.IntegrityState(status.Integrity),
		NTPDrift:          status.NTPDrift,
		NewEntriesBlocked: status.NewEntriesBlocked,
		HaltReason:        status.HaltReason,
		HaltedAtUTC:       haltedAt,
		Banner:            status.Banner,
		AnalyticsDegraded: status.AnalyticsDegraded,
		LastUpdatedAtUTC:  status.LastUpdatedAtUTC,
	}
}

// RunModeToRuntime converts a domain run mode into the runtime-layer value.
func RunModeToRuntime(mode entity.RunMode) entity.RunMode {
	return entity.RunMode(mode)
}

// ConnectivityToRuntime converts a domain connectivity state into the runtime-layer value.
func ConnectivityToRuntime(state entity.ConnectivityState) entity.ConnectivityState {
	return entity.ConnectivityState(state)
}

// LicenseToRuntime converts a domain license state into the runtime-layer value.
func LicenseToRuntime(state entity.LicenseState) entity.LicenseState {
	return entity.LicenseState(state)
}

func runModeFromRuntime(mode entity.RunMode) entity.RunMode {
	return entity.RunMode(mode)
}

func connectivityFromRuntime(state entity.ConnectivityState) entity.ConnectivityState {
	return entity.ConnectivityState(state)
}

func licenseFromRuntime(state entity.LicenseState) entity.LicenseState {
	return entity.LicenseState(state)
}

func integrityFromRuntime(state entity.IntegrityState) entity.IntegrityState {
	return entity.IntegrityState(state)
}
