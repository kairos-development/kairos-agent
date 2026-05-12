package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	stateDir := t.TempDir()
	cfg := Default(stateDir)

	assert.Equal(t, CurrentSchemaVersion, cfg.SchemaVersion)
	assert.Equal(t, "UTC", cfg.ExportTimezone)
	assert.Empty(t, cfg.TrustedPluginKeys)
	assert.Equal(t, TelemetryProfileMinimal, cfg.Telemetry.Profile)
	assert.True(t, cfg.Telemetry.Enabled)
	assert.Equal(t, int64(DefaultJournalMaxSizeBytes), cfg.JournalMaxSizeBytes)
	assert.Equal(t, DefaultMemoryDegradePercent, cfg.MemoryDegradeThresholdPct)
	assert.Equal(t, filepath.Join(stateDir, "config.yaml"), cfg.Paths.ConfigPath)
	assert.Equal(t, filepath.Join(stateDir, "trades.sqlite"), cfg.Paths.TradesDB)
	assert.Equal(t, filepath.Join(stateDir, "vault.db"), cfg.Paths.VaultDB)
	assert.Equal(t, filepath.Join(stateDir, "journal.log"), cfg.Paths.JournalPath)
	assert.Equal(t, filepath.Join(stateDir, "audit.log"), cfg.Paths.AuditPath)
	assert.Equal(t, "1000", cfg.Risk.MaxPosition)
	assert.Equal(t, "BTCUSDT", cfg.Exchange.DefaultSymbol)
	assert.Empty(t, cfg.Cloud.BaseURL)
	assert.Equal(t, "cloud.access_token", cfg.Cloud.AccessTokenRef)
	assert.False(t, cfg.Cloud.HandshakeEnabled)
}

func TestNewManager_CreatesDefaultConfig(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)
	require.NotNil(t, manager)

	cfg := manager.Current()
	require.NotNil(t, cfg)
	assert.Equal(t, CurrentSchemaVersion, cfg.SchemaVersion)
	assert.Equal(t, "UTC", cfg.ExportTimezone)

	// Verify config file was created
	configPath := filepath.Join(stateDir, "config.yaml")
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestNewManager_LoadsExistingConfig(t *testing.T) {
	stateDir := t.TempDir()
	configPath := filepath.Join(stateDir, "config.yaml")

	// Create a config file first
	cfg := Default(stateDir)
	cfg.ExportTimezone = "America/New_York"
	cfg.Risk.MaxPosition = "5000"
	manager := &Manager{path: configPath}
	err := manager.Write(&cfg)
	require.NoError(t, err)

	// Load it with NewManager
	manager, err = NewManager(stateDir)
	require.NoError(t, err)

	loaded := manager.Current()
	assert.Equal(t, "America/New_York", loaded.ExportTimezone)
	assert.Equal(t, "5000", loaded.Risk.MaxPosition)
}

func TestNewManager_InvalidConfig(t *testing.T) {
	stateDir := t.TempDir()
	configPath := filepath.Join(stateDir, "config.yaml")

	// Write invalid config
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0o600)
	require.NoError(t, err)

	_, err = NewManager(stateDir)
	assert.Error(t, err)
}

func TestManager_Current_ReturnsClone(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)

	cfg1 := manager.Current()
	cfg2 := manager.Current()

	// Verify they are different pointers
	assert.NotSame(t, cfg1, cfg2)

	// Modify one and verify the other is unchanged
	cfg1.TrustedPluginKeys = append(cfg1.TrustedPluginKeys, "test-key")
	assert.Empty(t, cfg2.TrustedPluginKeys)
}

func TestManager_Write(t *testing.T) {
	stateDir := t.TempDir()
	configPath := filepath.Join(stateDir, "config.yaml")
	manager := &Manager{path: configPath}

	cfg := Default(stateDir)
	cfg.ExportTimezone = "Europe/London"

	err := manager.Write(&cfg)
	require.NoError(t, err)

	// Verify file exists and can be read
	loaded, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "Europe/London", loaded.ExportTimezone)
}

func TestManager_Write_CreatesDirectory(t *testing.T) {
	stateDir := t.TempDir()
	nestedPath := filepath.Join(stateDir, "nested", "dir", "config.yaml")
	manager := &Manager{path: nestedPath}

	cfg := Default(stateDir)
	err := manager.Write(&cfg)
	require.NoError(t, err)

	_, err = os.Stat(nestedPath)
	assert.NoError(t, err)
}

func TestLoad_Success(t *testing.T) {
	stateDir := t.TempDir()
	configPath := filepath.Join(stateDir, "config.yaml")

	cfg := Default(stateDir)
	cfg.ExportTimezone = "Asia/Tokyo"
	manager := &Manager{path: configPath}
	err := manager.Write(&cfg)
	require.NoError(t, err)

	loaded, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "Asia/Tokyo", loaded.ExportTimezone)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoad_InvalidYAML(t *testing.T) {
	stateDir := t.TempDir()
	configPath := filepath.Join(stateDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: [[["), 0o600)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode config")
}

func TestLoad_ValidationFails(t *testing.T) {
	stateDir := t.TempDir()
	configPath := filepath.Join(stateDir, "config.yaml")

	cfg := Default(stateDir)
	cfg.SchemaVersion = 999 // Invalid version
	manager := &Manager{path: configPath}
	err := manager.Write(&cfg)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version")
}

func TestManager_ReloadFromDisk_Success(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)

	// Modify config on disk
	cfg := manager.Current()
	cfg.ExportTimezone = "Europe/Paris"
	err = manager.Write(cfg)
	require.NoError(t, err)

	// Reload
	result, err := manager.ReloadFromDisk()
	require.NoError(t, err)
	assert.True(t, result.Applied)
	assert.False(t, result.RequiresConfirm)
	assert.Empty(t, result.CriticalChanges)

	// Verify new config is active
	current := manager.Current()
	assert.Equal(t, "Europe/Paris", current.ExportTimezone)
}

func TestManager_ReloadFromDisk_CriticalChanges(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)

	// Modify critical field on disk
	cfg := manager.Current()
	oldMaxPosition := cfg.Risk.MaxPosition
	cfg.Risk.MaxPosition = "5000"
	err = manager.Write(cfg)
	require.NoError(t, err)

	// Reload
	result, err := manager.ReloadFromDisk()
	require.NoError(t, err)
	assert.False(t, result.Applied)
	assert.True(t, result.RequiresConfirm)
	assert.Contains(t, result.CriticalChanges, "risk.max_position")

	// Verify old config is still active
	current := manager.Current()
	assert.Equal(t, oldMaxPosition, current.Risk.MaxPosition)
}

func TestManager_ReloadFromDisk_InvalidConfig(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)

	// Write invalid config
	err = os.WriteFile(manager.path, []byte("invalid: yaml: [[["), 0o600)
	require.NoError(t, err)

	result, err := manager.ReloadFromDisk()
	assert.Error(t, err)
	assert.False(t, result.Applied)
}

func TestManager_Apply_Success(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)

	cfg := manager.Current()
	cfg.ExportTimezone = "Australia/Sydney"

	err = manager.Apply(cfg)
	require.NoError(t, err)

	// Verify config is active
	current := manager.Current()
	assert.Equal(t, "Australia/Sydney", current.ExportTimezone)

	// Verify config was persisted
	loaded, err := Load(manager.path)
	require.NoError(t, err)
	assert.Equal(t, "Australia/Sydney", loaded.ExportTimezone)
}

func TestManager_Apply_ValidationFails(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := NewManager(stateDir)
	require.NoError(t, err)

	cfg := manager.Current()
	oldTimezone := cfg.ExportTimezone
	cfg.ExportTimezone = "" // Invalid

	err = manager.Apply(cfg)
	assert.Error(t, err)

	// Verify old config is still active
	current := manager.Current()
	assert.Equal(t, oldTimezone, current.ExportTimezone)
}

func TestValidate_SchemaVersion(t *testing.T) {
	tests := []struct {
		name    string
		version int
		wantErr bool
	}{
		{"valid version", CurrentSchemaVersion, false},
		{"invalid version", 999, true},
		{"zero version", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			cfg.SchemaVersion = tt.version
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_ExportTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{"valid UTC", "UTC", false},
		{"valid America/New_York", "America/New_York", false},
		{"valid Europe/London", "Europe/London", false},
		{"invalid timezone", "Invalid/Timezone", true},
		{"empty timezone", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			cfg.ExportTimezone = tt.timezone
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_JournalMaxSizeBytes(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"valid size", 16 << 20, false},
		{"minimum size", 1, false},
		{"zero size", 0, true},
		{"negative size", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			cfg.JournalMaxSizeBytes = tt.size
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_MemoryDegradeThresholdPct(t *testing.T) {
	tests := []struct {
		name    string
		pct     int
		wantErr bool
	}{
		{"valid 90", 90, false},
		{"valid 1", 1, false},
		{"valid 100", 100, false},
		{"zero", 0, true},
		{"negative", -1, true},
		{"over 100", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			cfg.MemoryDegradeThresholdPct = tt.pct
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_TelemetryProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile TelemetryProfile
		wantErr bool
	}{
		{"off", TelemetryProfileOff, false},
		{"minimal", TelemetryProfileMinimal, false},
		{"support", TelemetryProfileSupport, false},
		{"sync", TelemetryProfileSync, false},
		{"invalid", TelemetryProfile("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			cfg.Telemetry.Profile = tt.profile
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_CloudConfig(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*Config)
		wantErr   bool
	}{
		{
			name: "disabled without base url",
		},
		{
			name: "valid https base url",
			configure: func(cfg *Config) {
				cfg.Cloud.BaseURL = "https://cloud.kairos.local"
				cfg.Cloud.HandshakeEnabled = true
			},
		},
		{
			name: "valid local http base url",
			configure: func(cfg *Config) {
				cfg.Cloud.BaseURL = "http://127.0.0.1:8080"
			},
		},
		{
			name: "invalid scheme",
			configure: func(cfg *Config) {
				cfg.Cloud.BaseURL = "ftp://cloud.kairos.local"
			},
			wantErr: true,
		},
		{
			name: "handshake requires base url",
			configure: func(cfg *Config) {
				cfg.Cloud.HandshakeEnabled = true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			if tt.configure != nil {
				tt.configure(&cfg)
			}
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTrustedFingerprints(t *testing.T) {
	tests := []struct {
		name         string
		fingerprints []string
		wantErr      bool
	}{
		{"empty list", []string{}, false},
		{"single valid", []string{"abc123"}, false},
		{"multiple valid", []string{"abc123", "def456"}, false},
		{"duplicate", []string{"abc123", "abc123"}, true},
		{"empty string", []string{""}, true},
		{"whitespace only", []string{"   "}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default(t.TempDir())
			cfg.TrustedPluginKeys = tt.fingerprints
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCriticalChanges(t *testing.T) {
	stateDir := t.TempDir()
	cfg1 := Default(stateDir)
	cfg2 := Default(stateDir)

	// No changes
	changes := cfg1.CriticalChanges(&cfg2)
	assert.Empty(t, changes)

	// Change risk.max_position
	cfg2.Risk.MaxPosition = "5000"
	changes = cfg1.CriticalChanges(&cfg2)
	assert.Contains(t, changes, "risk.max_position")

	// Change exchange.api_key_id
	cfg2 = Default(stateDir)
	cfg2.Exchange.APIKeyID = "new-key-id"
	changes = cfg1.CriticalChanges(&cfg2)
	assert.Contains(t, changes, "exchange.api_key_id")

	// Change exchange.default_symbol
	cfg2 = Default(stateDir)
	cfg2.Exchange.DefaultSymbol = "ETHUSDT"
	changes = cfg1.CriticalChanges(&cfg2)
	assert.Contains(t, changes, "exchange.default_symbol")

	// Change paths.vault_db
	cfg2 = Default(stateDir)
	cfg2.Paths.VaultDB = "/new/path/vault.db"
	changes = cfg1.CriticalChanges(&cfg2)
	assert.Contains(t, changes, "paths.vault_db")

	// Multiple critical changes
	cfg2 = Default(stateDir)
	cfg2.Risk.MaxPosition = "5000"
	cfg2.Exchange.DefaultSymbol = "ETHUSDT"
	changes = cfg1.CriticalChanges(&cfg2)
	assert.Len(t, changes, 2)
	assert.Contains(t, changes, "risk.max_position")
	assert.Contains(t, changes, "exchange.default_symbol")
}

func TestDiff(t *testing.T) {
	stateDir := t.TempDir()
	cfg1 := Default(stateDir)
	cfg2 := Default(stateDir)

	// No changes
	diff := Diff(&cfg1, &cfg2)
	assert.Empty(t, diff)

	// Change timezone
	cfg2.ExportTimezone = "America/New_York"
	diff = Diff(&cfg1, &cfg2)
	assert.Contains(t, diff, "- export_timezone: UTC")
	assert.Contains(t, diff, "+ export_timezone: America/New_York")

	// Nil configs
	diff = Diff(nil, &cfg2)
	assert.Empty(t, diff)

	diff = Diff(&cfg1, nil)
	assert.Empty(t, diff)
}

func TestTelemetryProfile_Valid(t *testing.T) {
	tests := []struct {
		name    string
		profile TelemetryProfile
		want    bool
	}{
		{"off", TelemetryProfileOff, true},
		{"minimal", TelemetryProfileMinimal, true},
		{"support", TelemetryProfileSupport, true},
		{"sync", TelemetryProfileSync, true},
		{"invalid", TelemetryProfile("invalid"), false},
		{"empty", TelemetryProfile(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.profile.Valid())
		})
	}
}
