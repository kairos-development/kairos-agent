package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFinancialRiskDisclaimer(t *testing.T) {
	assert.NotEmpty(t, FinancialRiskDisclaimer)
	assert.Contains(t, FinancialRiskDisclaimer, "risk")
	assert.Contains(t, FinancialRiskDisclaimer, "live trading")
}

func TestRunMode_Constants(t *testing.T) {
	assert.Equal(t, RunMode("idle"), RunModeIdle)
	assert.Equal(t, RunMode("scanning"), RunModeScanning)
	assert.Equal(t, RunMode("paper_trading"), RunModePaperTrading)
	assert.Equal(t, RunMode("backtesting"), RunModeBacktesting)
	assert.Equal(t, RunMode("optimizing"), RunModeOptimizing)
	assert.Equal(t, RunMode("live_trading"), RunModeLiveTrading)
}

func TestConnectivityState_Constants(t *testing.T) {
	assert.Equal(t, ConnectivityState("connected"), ConnectivityStateConnected)
	assert.Equal(t, ConnectivityState("reconnecting"), ConnectivityStateReconnecting)
	assert.Equal(t, ConnectivityState("network_waiting"), ConnectivityStateNetworkWait)
}

func TestLicenseState_Constants(t *testing.T) {
	assert.Equal(t, LicenseState("demo"), LicenseStateDemo)
	assert.Equal(t, LicenseState("licensed"), LicenseStateLicensed)
	assert.Equal(t, LicenseState("grace"), LicenseStateGrace)
	assert.Equal(t, LicenseState("risk_only"), LicenseStateRiskOnly)
	assert.Equal(t, LicenseState("unlicensed"), LicenseStateUnlicensed)
}

func TestIntegrityState_Constants(t *testing.T) {
	assert.Equal(t, IntegrityState("trusted"), IntegrityStateTrusted)
	assert.Equal(t, IntegrityState("degraded"), IntegrityStateDegraded)
}

func TestTelemetryProfile_Constants(t *testing.T) {
	assert.Equal(t, TelemetryProfile("off"), TelemetryProfileOff)
	assert.Equal(t, TelemetryProfile("minimal"), TelemetryProfileMinimal)
	assert.Equal(t, TelemetryProfile("support"), TelemetryProfileSupport)
	assert.Equal(t, TelemetryProfile("sync"), TelemetryProfileSync)
}

func TestRuntimeStatus_Fields(t *testing.T) {
	now := time.Now().UTC()
	status := RuntimeStatus{
		Mode:              RunModePaperTrading,
		Connectivity:      ConnectivityStateConnected,
		License:           LicenseStateDemo,
		Integrity:         IntegrityStateTrusted,
		NTPDrift:          50 * time.Millisecond,
		NewEntriesBlocked: false,
		Banner:            "Test Banner",
		AnalyticsDegraded: false,
		LastUpdatedAtUTC:  now,
	}

	assert.Equal(t, RunModePaperTrading, status.Mode)
	assert.Equal(t, ConnectivityStateConnected, status.Connectivity)
	assert.Equal(t, LicenseStateDemo, status.License)
	assert.Equal(t, IntegrityStateTrusted, status.Integrity)
	assert.Equal(t, 50*time.Millisecond, status.NTPDrift)
	assert.False(t, status.NewEntriesBlocked)
	assert.Equal(t, "Test Banner", status.Banner)
	assert.False(t, status.AnalyticsDegraded)
	assert.Equal(t, now, status.LastUpdatedAtUTC)
}

func TestStatePaths_Fields(t *testing.T) {
	paths := StatePaths{
		ConfigPath:  "/path/to/config.toml",
		TradesDB:    "/path/to/trades.db",
		VaultDB:     "/path/to/vault.db",
		JournalPath: "/path/to/journal",
		AuditPath:   "/path/to/audit.log",
	}

	assert.Equal(t, "/path/to/config.toml", paths.ConfigPath)
	assert.Equal(t, "/path/to/trades.db", paths.TradesDB)
	assert.Equal(t, "/path/to/vault.db", paths.VaultDB)
	assert.Equal(t, "/path/to/journal", paths.JournalPath)
	assert.Equal(t, "/path/to/audit.log", paths.AuditPath)
}

func TestRiskConfig_Fields(t *testing.T) {
	config := RiskConfig{
		MaxPosition: "1000.00",
	}

	assert.Equal(t, "1000.00", config.MaxPosition)
}

func TestExchangeConfig_Fields(t *testing.T) {
	config := ExchangeConfig{
		APIKeyID:      "key123",
		DefaultSymbol: "BTCUSDT",
	}

	assert.Equal(t, "key123", config.APIKeyID)
	assert.Equal(t, "BTCUSDT", config.DefaultSymbol)
}

func TestAgentConfig_Fields(t *testing.T) {
	paths := StatePaths{
		ConfigPath: "/config.toml",
		TradesDB:   "/trades.db",
	}

	risk := RiskConfig{
		MaxPosition: "1000.00",
	}

	exchange := ExchangeConfig{
		APIKeyID:      "key123",
		DefaultSymbol: "BTCUSDT",
	}

	config := AgentConfig{
		SchemaVersion:             1,
		ExportTimezone:            "UTC",
		TrustedPluginFingerprints: []string{"fp1", "fp2"},
		Telemetry: TelemetryConsent{
			Enabled:        true,
			Profile:        TelemetryProfileMinimal,
			ConsentVersion: "v1",
		},
		JournalMaxSizeBytes:       100 * 1024 * 1024,
		MemoryLimitBytes:          512 * 1024 * 1024,
		MemoryDegradeThresholdPct: 80,
		Paths:                     paths,
		Risk:                      risk,
		Exchange:                  exchange,
	}

	assert.Equal(t, 1, config.SchemaVersion)
	assert.Equal(t, "UTC", config.ExportTimezone)
	assert.Len(t, config.TrustedPluginFingerprints, 2)
	assert.True(t, config.Telemetry.Enabled)
	assert.Equal(t, TelemetryProfileMinimal, config.Telemetry.Profile)
	assert.Equal(t, int64(100*1024*1024), config.JournalMaxSizeBytes)
	assert.Equal(t, int64(512*1024*1024), config.MemoryLimitBytes)
	assert.Equal(t, 80, config.MemoryDegradeThresholdPct)
	assert.Equal(t, "/config.toml", config.Paths.ConfigPath)
	assert.Equal(t, "1000.00", config.Risk.MaxPosition)
	assert.Equal(t, "BTCUSDT", config.Exchange.DefaultSymbol)
}

func TestTrustedPluginKey_Fields(t *testing.T) {
	now := time.Now().UTC()
	key := TrustedPluginKey{
		Fingerprint: "abc123",
		PublicKey:   []byte("public-key-data"),
		AddedAtUTC:  now,
	}

	assert.Equal(t, "abc123", key.Fingerprint)
	assert.Equal(t, []byte("public-key-data"), key.PublicKey)
	assert.Equal(t, now, key.AddedAtUTC)
}

func TestRuntimeStatus_AllModes(t *testing.T) {
	modes := []RunMode{
		RunModeIdle,
		RunModeScanning,
		RunModePaperTrading,
		RunModeBacktesting,
		RunModeOptimizing,
		RunModeLiveTrading,
	}

	for _, mode := range modes {
		status := RuntimeStatus{Mode: mode}
		assert.NotEmpty(t, status.Mode)
	}
}

func TestRuntimeStatus_AllConnectivityStates(t *testing.T) {
	states := []ConnectivityState{
		ConnectivityStateConnected,
		ConnectivityStateReconnecting,
		ConnectivityStateNetworkWait,
	}

	for _, state := range states {
		status := RuntimeStatus{Connectivity: state}
		assert.NotEmpty(t, status.Connectivity)
	}
}

func TestRuntimeStatus_AllLicenseStates(t *testing.T) {
	states := []LicenseState{
		LicenseStateDemo,
		LicenseStateLicensed,
		LicenseStateGrace,
		LicenseStateRiskOnly,
		LicenseStateUnlicensed,
	}

	for _, state := range states {
		status := RuntimeStatus{License: state}
		assert.NotEmpty(t, status.License)
	}
}

func TestRuntimeStatus_AllIntegrityStates(t *testing.T) {
	states := []IntegrityState{
		IntegrityStateTrusted,
		IntegrityStateDegraded,
	}

	for _, state := range states {
		status := RuntimeStatus{Integrity: state}
		assert.NotEmpty(t, status.Integrity)
	}
}

func TestAgentConfig_EmptyTrustedPlugins(t *testing.T) {
	config := AgentConfig{
		TrustedPluginFingerprints: []string{},
	}

	assert.Empty(t, config.TrustedPluginFingerprints)
}

func TestAgentConfig_MultipleTrustedPlugins(t *testing.T) {
	config := AgentConfig{
		TrustedPluginFingerprints: []string{"fp1", "fp2", "fp3"},
	}

	assert.Len(t, config.TrustedPluginFingerprints, 3)
	assert.Contains(t, config.TrustedPluginFingerprints, "fp1")
	assert.Contains(t, config.TrustedPluginFingerprints, "fp2")
	assert.Contains(t, config.TrustedPluginFingerprints, "fp3")
}
