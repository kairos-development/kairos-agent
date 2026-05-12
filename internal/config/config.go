package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kairos-development/kairos-connectors/pkg/connectors/bybit"
	"gopkg.in/yaml.v3"
)

const (
	CurrentSchemaVersion        = 1
	DefaultJournalMaxSizeBytes  = 16 << 20
	DefaultMemoryDegradePercent = 90
)

// PathsConfig describes stateful file locations relative to the state directory.
type PathsConfig struct {
	ConfigPath  string `yaml:"config_path"`
	TradesDB    string `yaml:"trades_db"`
	VaultDB     string `yaml:"vault_db"`
	JournalPath string `yaml:"journal_path"`
	AuditPath   string `yaml:"audit_path"`
}

// RiskConfig captures operator risk configuration.
type RiskConfig struct {
	MaxPosition string `yaml:"max_position"`
}

// ExchangeConfig captures cold-start exchange configuration.
type ExchangeConfig struct {
	APIKeyID      string `yaml:"api_key_id"`
	DefaultSymbol string `yaml:"default_symbol"`
}

// CloudConfig captures non-secret Kairos Cloud integration settings.
type CloudConfig struct {
	BaseURL          string `yaml:"base_url"`
	AccessTokenRef   string `yaml:"access_token_ref"`
	HandshakeEnabled bool   `yaml:"handshake_enabled"`
}

// Config is the top-level agent configuration file.
type Config struct {
	SchemaVersion             int               `yaml:"schema_version"`
	ExportTimezone            string            `yaml:"export_timezone"`
	TrustedPluginKeys         []string          `yaml:"trusted_plugin_keys"`
	Telemetry                 TelemetryConsent  `yaml:"telemetry"`
	JournalMaxSizeBytes       int64             `yaml:"journal_max_size_bytes"`
	MemoryLimitBytes          int64             `yaml:"memory_limit_bytes"`
	MemoryDegradeThresholdPct int               `yaml:"memory_degrade_threshold_percent"`
	Proxy                     bybit.ProxyConfig `yaml:"proxy"`
	Paths                     PathsConfig       `yaml:"paths"`
	Risk                      RiskConfig        `yaml:"risk"`
	Exchange                  ExchangeConfig    `yaml:"exchange"`
	Cloud                     CloudConfig       `yaml:"cloud"`
	Edition                   string            `yaml:"edition"`
	LicenseExpiry             int64             `yaml:"license_expiry"`
	GraceStartedAt            *time.Time        `yaml:"grace_started_at"`
}

// Manager atomically owns the active configuration.
type Manager struct {
	path string
	live atomic.Pointer[Config]
}

// ReloadResult describes the outcome of a reload attempt.
type ReloadResult struct {
	Applied         bool
	RequiresConfirm bool
	CriticalChanges []string
	Diff            string
}

// Default returns a baseline config for the provided state directory.
func Default(stateDir string) Config {
	return Config{
		SchemaVersion:     CurrentSchemaVersion,
		ExportTimezone:    "UTC",
		TrustedPluginKeys: []string{},
		Telemetry: TelemetryConsent{
			Enabled:        true,
			Profile:        TelemetryProfileMinimal,
			ConsentVersion: "v1",
		},
		Edition:                   "community",
		LicenseExpiry:             0,
		JournalMaxSizeBytes:       DefaultJournalMaxSizeBytes,
		MemoryDegradeThresholdPct: DefaultMemoryDegradePercent,
		Paths: PathsConfig{
			ConfigPath:  filepath.Join(stateDir, "config.yaml"),
			TradesDB:    filepath.Join(stateDir, "trades.sqlite"),
			VaultDB:     filepath.Join(stateDir, "vault.db"),
			JournalPath: filepath.Join(stateDir, "journal.log"),
			AuditPath:   filepath.Join(stateDir, "audit.log"),
		},
		Risk:     RiskConfig{MaxPosition: "1000"},
		Exchange: ExchangeConfig{DefaultSymbol: "BTCUSDT"},
		Cloud: CloudConfig{
			BaseURL:          "",
			AccessTokenRef:   "cloud.access_token",
			HandshakeEnabled: false,
		},
	}
}

// NewManager loads or initializes configuration management for the given state directory.
func NewManager(stateDir string) (*Manager, error) {
	defaultConfig := Default(stateDir)
	manager := &Manager{path: defaultConfig.Paths.ConfigPath}
	if _, err := os.Stat(manager.path); err == nil {
		cfg, err := Load(manager.path)
		if err != nil {
			return nil, err
		}
		manager.live.Store(cfg)
		return manager, nil
	}
	if err := manager.Write(&defaultConfig); err != nil {
		return nil, err
	}
	manager.live.Store(&defaultConfig)
	return manager, nil
}

// Current returns the active config snapshot.
func (m *Manager) Current() *Config {
	cfg := m.live.Load()
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.TrustedPluginKeys = append([]string(nil), cfg.TrustedPluginKeys...)
	return &clone
}

// Write persists a config to disk.
func (m *Manager) Write(cfg *Config) error {
	body, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(m.path, body, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Load reads and validates a config file.
func Load(path string) (*Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ReloadFromDisk validates the on-disk config and atomically swaps it into place.
func (m *Manager) ReloadFromDisk() (ReloadResult, error) {
	current := m.Current()
	candidate, err := Load(m.path)
	if err != nil {
		return ReloadResult{Applied: false, Diff: err.Error()}, err
	}
	result := ReloadResult{CriticalChanges: current.CriticalChanges(candidate), Diff: Diff(current, candidate)}
	if len(result.CriticalChanges) > 0 {
		result.RequiresConfirm = true
		return result, nil
	}
	m.live.Store(candidate)
	result.Applied = true
	return result, nil
}

// Apply atomically swaps a validated config into place and persists it.
func (m *Manager) Apply(cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := m.Write(cfg); err != nil {
		return err
	}
	m.live.Store(cfg)
	return nil
}

// Validate enforces config invariants.
func (c *Config) Validate() error {
	if c.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("schema_version %d is incompatible with current version %d", c.SchemaVersion, CurrentSchemaVersion)
	}
	if c.ExportTimezone == "" {
		return fmt.Errorf("export_timezone is required")
	}
	if _, err := time.LoadLocation(c.ExportTimezone); err != nil {
		return fmt.Errorf("invalid export_timezone %q: %w", c.ExportTimezone, err)
	}
	if c.JournalMaxSizeBytes <= 0 {
		return fmt.Errorf("journal_max_size_bytes must be positive")
	}
	if c.MemoryDegradeThresholdPct < 1 || c.MemoryDegradeThresholdPct > 100 {
		return fmt.Errorf("memory_degrade_threshold_percent must be between 1 and 100")
	}
	if !c.Telemetry.Profile.Valid() {
		return fmt.Errorf("invalid telemetry_profile %q", c.Telemetry.Profile)
	}
	if err := validateTrustedFingerprints(c.TrustedPluginKeys); err != nil {
		return err
	}
	if c.Cloud.BaseURL != "" {
		parsed, err := url.ParseRequestURI(c.Cloud.BaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid cloud.base_url %q", c.Cloud.BaseURL)
		}
		switch parsed.Scheme {
		case "http", "https":
		default:
			return fmt.Errorf("cloud.base_url must use http or https")
		}
	}
	if c.Cloud.HandshakeEnabled && c.Cloud.BaseURL == "" {
		return fmt.Errorf("cloud.base_url is required when cloud.handshake_enabled is true")
	}
	return nil
}

// CriticalChanges reports critical fields that differ between two configs.
func (c *Config) CriticalChanges(other *Config) []string {
	var changes []string
	if c.Risk.MaxPosition != other.Risk.MaxPosition {
		changes = append(changes, "risk.max_position")
	}
	if c.Exchange.APIKeyID != other.Exchange.APIKeyID {
		changes = append(changes, "exchange.api_key_id")
	}
	if c.Exchange.DefaultSymbol != other.Exchange.DefaultSymbol {
		changes = append(changes, "exchange.default_symbol")
	}
	if c.Paths.VaultDB != other.Paths.VaultDB {
		changes = append(changes, "paths.vault_db")
	}
	return changes
}

func validateTrustedFingerprints(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("trusted plugin key fingerprint cannot be empty")
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("duplicate trusted plugin key fingerprint %q", value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

// Diff produces a compact line-oriented config diff.
func Diff(current *Config, candidate *Config) string {
	if current == nil || candidate == nil {
		return ""
	}
	left, _ := yaml.Marshal(current)
	right, _ := yaml.Marshal(candidate)
	leftLines := strings.Split(strings.TrimSpace(string(left)), "\n")
	rightLines := strings.Split(strings.TrimSpace(string(right)), "\n")
	var buf bytes.Buffer
	max := len(leftLines)
	if len(rightLines) > max {
		max = len(rightLines)
	}
	for i := 0; i < max; i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		if l == r {
			continue
		}
		if l != "" {
			buf.WriteString("- ")
			buf.WriteString(l)
			buf.WriteByte('\n')
		}
		if r != "" {
			buf.WriteString("+ ")
			buf.WriteString(r)
			buf.WriteByte('\n')
		}
	}
	return strings.TrimSpace(buf.String())
}
