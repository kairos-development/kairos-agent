package entity

import "time"

// FinancialRiskDisclaimer is the domain disclaimer required before live trading.
const FinancialRiskDisclaimer = "Kairos is an execution and automation platform, not financial advice. Use of live trading is at your own risk."

// RunMode identifies the active runtime mode in domain terms.
type RunMode string

const (
	RunModeIdle         RunMode = "idle"
	RunModeScanning     RunMode = "scanning"
	RunModePaperTrading RunMode = "paper_trading"
	RunModeBacktesting  RunMode = "backtesting"
	RunModeOptimizing   RunMode = "optimizing"
	RunModeLiveTrading  RunMode = "live_trading"
	RunModeHalted       RunMode = "halted"
)

// Display returns a human-readable representation of the run mode.
func (m RunMode) Display() string {
	switch m {
	case RunModeIdle:
		return "Idle"
	case RunModeScanning:
		return "Scanning"
	case RunModePaperTrading:
		return "Paper Trading"
	case RunModeBacktesting:
		return "Backtesting"
	case RunModeOptimizing:
		return "Optimizing"
	case RunModeLiveTrading:
		return "Live Trading"
	case RunModeHalted:
		return "Halted"
	default:
		return string(m)
	}
}

// ConnectivityState identifies the exchange connectivity state.
type ConnectivityState string

const (
	ConnectivityStateConnected    ConnectivityState = "connected"
	ConnectivityStateReconnecting ConnectivityState = "reconnecting"
	ConnectivityStateNetworkWait  ConnectivityState = "network_waiting"
)

// LicenseState identifies the local license state.
type LicenseState string

const (
	LicenseStateDemo       LicenseState = "demo"
	LicenseStateLicensed   LicenseState = "licensed"
	LicenseStateGrace      LicenseState = "grace"
	LicenseStateRiskOnly   LicenseState = "risk_only"
	LicenseStateUnlicensed LicenseState = "unlicensed"
)

// IntegrityState identifies runtime integrity status.
type IntegrityState string

const (
	IntegrityStateTrusted  IntegrityState = "trusted"
	IntegrityStateDegraded IntegrityState = "degraded"
)

// TelemetryProfile identifies the operator-approved telemetry profile.
type TelemetryProfile string

const (
	TelemetryProfileOff     TelemetryProfile = "off"
	TelemetryProfileMinimal TelemetryProfile = "minimal"
	TelemetryProfileSupport TelemetryProfile = "support"
	TelemetryProfileSync    TelemetryProfile = "sync"
)

// TelemetryConsent records the operator's explicit telemetry consent decision.
type TelemetryConsent struct {
	Enabled        bool
	Profile        TelemetryProfile
	ConsentVersion string
	ConsentSource  string
	ConsentedAtUTC time.Time
	InstallID      string
}

// RuntimeStatus describes the domain-visible agent runtime state.
type RuntimeStatus struct {
	Mode              RunMode
	Connectivity      ConnectivityState
	License           LicenseState
	Integrity         IntegrityState
	NTPDrift          time.Duration
	NewEntriesBlocked bool
	HaltReason        string
	HaltedAtUTC       *time.Time
	Banner            string
	AnalyticsDegraded bool
	LastUpdatedAtUTC  time.Time
}

// StatePaths contains the resolved state file locations.
type StatePaths struct {
	ConfigPath  string
	TradesDB    string
	VaultDB     string
	JournalPath string
	AuditPath   string
}

// RiskConfig contains operator risk settings relevant to the domain layer.
type RiskConfig struct {
	MaxPosition string
}

// ExchangeConfig contains exchange bootstrap settings relevant to the domain layer.
type ExchangeConfig struct {
	APIKeyID      string
	DefaultSymbol string
}

// AgentConfig contains the domain-visible operator configuration.
type AgentConfig struct {
	SchemaVersion             int
	ExportTimezone            string
	TrustedPluginFingerprints []string
	Telemetry                 TelemetryConsent
	JournalMaxSizeBytes       int64
	MemoryLimitBytes          int64
	MemoryDegradeThresholdPct int
	Paths                     StatePaths
	Risk                      RiskConfig
	Exchange                  ExchangeConfig
	Edition                   string
	LicenseExpiry             int64
	GraceStartedAt            *time.Time
}

// TrustedPluginKey stores a trusted plugin signing key.
type TrustedPluginKey struct {
	Fingerprint string
	PublicKey   []byte
	AddedAtUTC  time.Time
}

// Limits describes numerical limits for the current edition.
type Limits struct {
	MaxExchanges  int
	MaxStrategies int
	MaxConnectors int
}
