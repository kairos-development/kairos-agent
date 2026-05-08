package conv

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	"github.com/stretchr/testify/assert"
)

func TestConfigToDomain(t *testing.T) {
	cfg := &config.Config{
		SchemaVersion:             1,
		ExportTimezone:            "America/New_York",
		TrustedPluginKeys:         []string{"key1", "key2"},
		Telemetry:                 config.TelemetryConsent{Enabled: true, Profile: config.TelemetryProfileMinimal, ConsentVersion: "v1"},
		JournalMaxSizeBytes:       32 << 20,
		MemoryLimitBytes:          2 << 30,
		MemoryDegradeThresholdPct: 85,
		Paths: config.PathsConfig{
			ConfigPath:  "/state/config.yaml",
			TradesDB:    "/state/trades.sqlite",
			VaultDB:     "/state/vault.db",
			JournalPath: "/state/journal.log",
			AuditPath:   "/state/audit.log",
		},
		Risk: config.RiskConfig{
			MaxPosition: "5000",
		},
		Exchange: config.ExchangeConfig{
			APIKeyID:      "test-key",
			DefaultSymbol: "BTCUSDT",
		},
	}

	result := ConfigToDomain(cfg)

	assert.Equal(t, 1, result.SchemaVersion)
	assert.Equal(t, "America/New_York", result.ExportTimezone)
	assert.Equal(t, []string{"key1", "key2"}, result.TrustedPluginFingerprints)
	assert.Equal(t, entity.TelemetryProfileMinimal, result.Telemetry.Profile)
	assert.True(t, result.Telemetry.Enabled)
	assert.Equal(t, int64(32<<20), result.JournalMaxSizeBytes)
	assert.Equal(t, int64(2<<30), result.MemoryLimitBytes)
	assert.Equal(t, 85, result.MemoryDegradeThresholdPct)
	assert.Equal(t, "/state/config.yaml", result.Paths.ConfigPath)
	assert.Equal(t, "/state/trades.sqlite", result.Paths.TradesDB)
	assert.Equal(t, "/state/vault.db", result.Paths.VaultDB)
	assert.Equal(t, "/state/journal.log", result.Paths.JournalPath)
	assert.Equal(t, "/state/audit.log", result.Paths.AuditPath)
	assert.Equal(t, "5000", result.Risk.MaxPosition)
	assert.Equal(t, "test-key", result.Exchange.APIKeyID)
	assert.Equal(t, "BTCUSDT", result.Exchange.DefaultSymbol)

	cfg.TrustedPluginKeys[0] = "modified"
	assert.Equal(t, "key1", result.TrustedPluginFingerprints[0], "should be a copy, not shared")
}

func TestConfigFromDomain(t *testing.T) {
	domainCfg := entity.AgentConfig{
		SchemaVersion:             1,
		ExportTimezone:            "Europe/London",
		TrustedPluginFingerprints: []string{"fp1", "fp2"},
		Telemetry:                 entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileSync},
		JournalMaxSizeBytes:       64 << 20,
		MemoryLimitBytes:          4 << 30,
		MemoryDegradeThresholdPct: 90,
		Paths: entity.StatePaths{
			ConfigPath:  "/data/config.yaml",
			TradesDB:    "/data/trades.sqlite",
			VaultDB:     "/data/vault.db",
			JournalPath: "/data/journal.log",
			AuditPath:   "/data/audit.log",
		},
		Risk: entity.RiskConfig{
			MaxPosition: "10000",
		},
		Exchange: entity.ExchangeConfig{
			APIKeyID:      "prod-key",
			DefaultSymbol: "ETHUSDT",
		},
	}

	result := ConfigFromDomain(domainCfg)

	assert.Equal(t, 1, result.SchemaVersion)
	assert.Equal(t, "Europe/London", result.ExportTimezone)
	assert.Equal(t, []string{"fp1", "fp2"}, result.TrustedPluginKeys)
	assert.Equal(t, config.TelemetryProfileSync, result.Telemetry.Profile)
	assert.True(t, result.Telemetry.Enabled)
	assert.Equal(t, int64(64<<20), result.JournalMaxSizeBytes)
	assert.Equal(t, int64(4<<30), result.MemoryLimitBytes)
	assert.Equal(t, 90, result.MemoryDegradeThresholdPct)
	assert.Equal(t, "/data/config.yaml", result.Paths.ConfigPath)
	assert.Equal(t, "/data/trades.sqlite", result.Paths.TradesDB)
	assert.Equal(t, "/data/vault.db", result.Paths.VaultDB)
	assert.Equal(t, "/data/journal.log", result.Paths.JournalPath)
	assert.Equal(t, "/data/audit.log", result.Paths.AuditPath)
	assert.Equal(t, "10000", result.Risk.MaxPosition)
	assert.Equal(t, "prod-key", result.Exchange.APIKeyID)
	assert.Equal(t, "ETHUSDT", result.Exchange.DefaultSymbol)

	domainCfg.TrustedPluginFingerprints[0] = "modified"
	assert.Equal(t, "fp1", result.TrustedPluginKeys[0], "should be a copy, not shared")
}

func TestConfigToDomain_EmptyTrustedKeys(t *testing.T) {
	cfg := &config.Config{
		SchemaVersion:     1,
		ExportTimezone:    "UTC",
		TrustedPluginKeys: []string{},
		Telemetry:         config.TelemetryConsent{Enabled: false, Profile: config.TelemetryProfileOff},
		Paths: config.PathsConfig{
			ConfigPath:  "/state/config.yaml",
			TradesDB:    "/state/trades.sqlite",
			VaultDB:     "/state/vault.db",
			JournalPath: "/state/journal.log",
			AuditPath:   "/state/audit.log",
		},
	}

	result := ConfigToDomain(cfg)
	assert.Empty(t, result.TrustedPluginFingerprints)
}

func TestRuntimeStatusToDomain(t *testing.T) {
	now := time.Now().UTC()
	status := runtime.Status{
		Mode:              entity.RunModeLiveTrading,
		Connectivity:      entity.ConnectivityStateConnected,
		License:           entity.LicenseStateLicensed,
		Integrity:         entity.IntegrityStateTrusted,
		NTPDrift:          12 * time.Millisecond,
		NewEntriesBlocked: false,
		Banner:            "Test Banner",
		AnalyticsDegraded: true,
		LastUpdatedAtUTC:  now,
	}

	result := RuntimeStatusToDomain(status)

	assert.Equal(t, entity.RunModeLiveTrading, result.Mode)
	assert.Equal(t, entity.ConnectivityStateConnected, result.Connectivity)
	assert.Equal(t, entity.LicenseStateLicensed, result.License)
	assert.Equal(t, entity.IntegrityStateTrusted, result.Integrity)
	assert.Equal(t, 12*time.Millisecond, result.NTPDrift)
	assert.False(t, result.NewEntriesBlocked)
	assert.Equal(t, "Test Banner", result.Banner)
	assert.True(t, result.AnalyticsDegraded)
	assert.Equal(t, now, result.LastUpdatedAtUTC)
}

func TestRunModeToRuntime(t *testing.T) {
	tests := []struct {
		domain   entity.RunMode
		expected entity.RunMode
	}{
		{entity.RunModeIdle, entity.RunModeIdle},
		{entity.RunModeScanning, entity.RunModeScanning},
		{entity.RunModePaperTrading, entity.RunModePaperTrading},
		{entity.RunModeLiveTrading, entity.RunModeLiveTrading},
		{entity.RunModeBacktesting, entity.RunModeBacktesting},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			result := RunModeToRuntime(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConnectivityToRuntime(t *testing.T) {
	tests := []struct {
		domain   entity.ConnectivityState
		expected entity.ConnectivityState
	}{
		{entity.ConnectivityStateConnected, entity.ConnectivityStateConnected},
		{entity.ConnectivityStateReconnecting, entity.ConnectivityStateReconnecting},
		{entity.ConnectivityStateNetworkWait, entity.ConnectivityStateNetworkWait},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			result := ConnectivityToRuntime(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLicenseToRuntime(t *testing.T) {
	tests := []struct {
		domain   entity.LicenseState
		expected entity.LicenseState
	}{
		{entity.LicenseStateDemo, entity.LicenseStateDemo},
		{entity.LicenseStateLicensed, entity.LicenseStateLicensed},
		{entity.LicenseStateGrace, entity.LicenseStateGrace},
		{entity.LicenseStateRiskOnly, entity.LicenseStateRiskOnly},
		{entity.LicenseStateUnlicensed, entity.LicenseStateUnlicensed},
	}

	for _, tt := range tests {
		t.Run(string(tt.domain), func(t *testing.T) {
			result := LicenseToRuntime(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTelemetryProfileConversions(t *testing.T) {
	tests := []struct {
		configProfile config.TelemetryProfile
		domainProfile entity.TelemetryProfile
	}{
		{config.TelemetryProfileOff, entity.TelemetryProfileOff},
		{config.TelemetryProfileMinimal, entity.TelemetryProfileMinimal},
		{config.TelemetryProfileSupport, entity.TelemetryProfileSupport},
		{config.TelemetryProfileSync, entity.TelemetryProfileSync},
	}

	for _, tt := range tests {
		t.Run(string(tt.configProfile), func(t *testing.T) {
			assert.Equal(t, tt.domainProfile, entity.TelemetryProfile(tt.configProfile))
			assert.Equal(t, tt.configProfile, config.TelemetryProfile(tt.domainProfile))
		})
	}
}

func TestRuntimeConversions(t *testing.T) {
	t.Run("runModeFromRuntime", func(t *testing.T) {
		modes := []entity.RunMode{
			entity.RunModeIdle,
			entity.RunModeScanning,
			entity.RunModePaperTrading,
			entity.RunModeLiveTrading,
			entity.RunModeBacktesting,
		}
		for _, mode := range modes {
			result := runModeFromRuntime(mode)
			assert.Equal(t, entity.RunMode(mode), result)
		}
	})

	t.Run("connectivityFromRuntime", func(t *testing.T) {
		states := []entity.ConnectivityState{
			entity.ConnectivityStateConnected,
			entity.ConnectivityStateReconnecting,
			entity.ConnectivityStateNetworkWait,
		}
		for _, state := range states {
			result := connectivityFromRuntime(state)
			assert.Equal(t, entity.ConnectivityState(state), result)
		}
	})

	t.Run("licenseFromRuntime", func(t *testing.T) {
		states := []entity.LicenseState{
			entity.LicenseStateDemo,
			entity.LicenseStateLicensed,
			entity.LicenseStateGrace,
			entity.LicenseStateRiskOnly,
			entity.LicenseStateUnlicensed,
		}
		for _, state := range states {
			result := licenseFromRuntime(state)
			assert.Equal(t, entity.LicenseState(state), result)
		}
	})

	t.Run("integrityFromRuntime", func(t *testing.T) {
		states := []entity.IntegrityState{
			entity.IntegrityStateTrusted,
			entity.IntegrityStateDegraded,
		}
		for _, state := range states {
			result := integrityFromRuntime(state)
			assert.Equal(t, entity.IntegrityState(state), result)
		}
	})
}
