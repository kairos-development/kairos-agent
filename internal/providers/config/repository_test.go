package configprovider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Config(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	cfg, err := repo.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, config.CurrentSchemaVersion, cfg.SchemaVersion)
	assert.Equal(t, "UTC", cfg.ExportTimezone)
}

func TestRepository_SetConfig(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	newConfig := entity.AgentConfig{
		SchemaVersion:             1,
		ExportTimezone:            "America/New_York",
		TrustedPluginFingerprints: []string{"test-fingerprint"},
		Telemetry:                 entity.TelemetryConsent{Enabled: false, Profile: entity.TelemetryProfileOff},
		JournalMaxSizeBytes:       32 << 20,
		MemoryLimitBytes:          1 << 30,
		MemoryDegradeThresholdPct: 85,
		Paths: entity.StatePaths{
			ConfigPath:  filepath.Join(stateDir, "config.yaml"),
			TradesDB:    filepath.Join(stateDir, "trades.sqlite"),
			VaultDB:     filepath.Join(stateDir, "vault.db"),
			JournalPath: filepath.Join(stateDir, "journal.log"),
			AuditPath:   filepath.Join(stateDir, "audit.log"),
		},
		Risk: entity.RiskConfig{
			MaxPosition: "5000",
		},
		Exchange: entity.ExchangeConfig{
			APIKeyID:      "test-key",
			DefaultSymbol: "ETHUSDT",
		},
	}

	err = repo.SetConfig(ctx, newConfig)
	require.NoError(t, err)

	retrieved, err := repo.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", retrieved.ExportTimezone)
	assert.Equal(t, int64(32<<20), retrieved.JournalMaxSizeBytes)
	assert.Equal(t, "5000", retrieved.Risk.MaxPosition)
	assert.Equal(t, "ETHUSDT", retrieved.Exchange.DefaultSymbol)
	assert.Equal(t, []string{"test-fingerprint"}, retrieved.TrustedPluginFingerprints)
}

func TestRepository_SetConfig_InvalidSchema(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	invalidConfig := entity.AgentConfig{
		SchemaVersion:  999,
		ExportTimezone: "UTC",
		Paths: entity.StatePaths{
			ConfigPath:  filepath.Join(stateDir, "config.yaml"),
			TradesDB:    filepath.Join(stateDir, "trades.sqlite"),
			VaultDB:     filepath.Join(stateDir, "vault.db"),
			JournalPath: filepath.Join(stateDir, "journal.log"),
			AuditPath:   filepath.Join(stateDir, "audit.log"),
		},
	}

	err = repo.SetConfig(ctx, invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version")
}

func TestRepository_UpdateConfig_StringFields(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	tests := []struct {
		key      string
		value    interface{}
		validate func(t *testing.T, cfg entity.AgentConfig)
	}{
		{
			key:   "export_timezone",
			value: "Europe/London",
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, "Europe/London", cfg.ExportTimezone)
			},
		},
		{
			key:   "risk.max_position",
			value: "2500",
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, "2500", cfg.Risk.MaxPosition)
			},
		},
		{
			key:   "exchange.default_symbol",
			value: "SOLUSDT",
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, "SOLUSDT", cfg.Exchange.DefaultSymbol)
			},
		},
		{
			key:   "exchange.api_key_id",
			value: "new-api-key",
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, "new-api-key", cfg.Exchange.APIKeyID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			updates := map[string]interface{}{tt.key: tt.value}
			err := repo.UpdateConfig(ctx, updates)
			require.NoError(t, err)

			cfg, err := repo.Config(ctx)
			require.NoError(t, err)
			tt.validate(t, cfg)
		})
	}
}

func TestRepository_UpdateConfig_IntFields(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	tests := []struct {
		key      string
		value    interface{}
		validate func(t *testing.T, cfg entity.AgentConfig)
	}{
		{
			key:   "journal_max_size_bytes",
			value: int64(64 << 20),
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, int64(64<<20), cfg.JournalMaxSizeBytes)
			},
		},
		{
			key:   "memory_limit_bytes",
			value: int64(2 << 30),
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, int64(2<<30), cfg.MemoryLimitBytes)
			},
		},
		{
			key:   "memory_degrade_threshold_percent",
			value: 80,
			validate: func(t *testing.T, cfg entity.AgentConfig) {
				assert.Equal(t, 80, cfg.MemoryDegradeThresholdPct)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			updates := map[string]interface{}{tt.key: tt.value}
			err := repo.UpdateConfig(ctx, updates)
			require.NoError(t, err)

			cfg, err := repo.Config(ctx)
			require.NoError(t, err)
			tt.validate(t, cfg)
		})
	}
}

func TestRepository_UpdateConfig_TelemetryProfile(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	updates := map[string]interface{}{"telemetry.profile": "minimal"}
	err = repo.UpdateConfig(ctx, updates)
	require.NoError(t, err)

	cfg, err := repo.Config(ctx)
	require.NoError(t, err)
	assert.True(t, cfg.Telemetry.Enabled)
	assert.Equal(t, entity.TelemetryProfileMinimal, cfg.Telemetry.Profile)
}

func TestRepository_UpdateConfig_UnknownKey(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	updates := map[string]interface{}{"unknown_key": "value"}
	err = repo.UpdateConfig(ctx, updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown configuration key")
}

func TestRepository_UpdateConfig_WrongType(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	updates := map[string]interface{}{"export_timezone": 123}
	err = repo.UpdateConfig(ctx, updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected string")
}

func TestRepository_UpdateConfig_MultipleFields(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	updates := map[string]interface{}{
		"export_timezone":        "Asia/Tokyo",
		"risk.max_position":      "3000",
		"journal_max_size_bytes": int64(128 << 20),
	}
	err = repo.UpdateConfig(ctx, updates)
	require.NoError(t, err)

	cfg, err := repo.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Asia/Tokyo", cfg.ExportTimezone)
	assert.Equal(t, "3000", cfg.Risk.MaxPosition)
	assert.Equal(t, int64(128<<20), cfg.JournalMaxSizeBytes)
}

func TestRepository_UpdateConfig_Persistence(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	updates := map[string]interface{}{"risk.max_position": "7500"}
	err = repo.UpdateConfig(ctx, updates)
	require.NoError(t, err)

	configPath := filepath.Join(stateDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "max_position: \"7500\"")

	newManager, err := config.NewManager(stateDir)
	require.NoError(t, err)
	newRepo := New(newManager)

	cfg, err := newRepo.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, "7500", cfg.Risk.MaxPosition)
}

func TestRepository_UpdateConfig_IntConversion(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	updates := map[string]interface{}{
		"journal_max_size_bytes": int(128 << 20),
		"memory_limit_bytes":     int(4 << 30),
	}
	err = repo.UpdateConfig(ctx, updates)
	require.NoError(t, err)

	cfg, err := repo.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(128<<20), cfg.JournalMaxSizeBytes)
	assert.Equal(t, int64(4<<30), cfg.MemoryLimitBytes)
}

func TestRepository_UpdateConfig_TypeErrors(t *testing.T) {
	stateDir := t.TempDir()
	manager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	repo := New(manager)
	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		value   interface{}
		errText string
	}{
		{
			name:    "telemetry_profile_wrong_type",
			key:     "telemetry.profile",
			value:   123,
			errText: "expected string",
		},
		{
			name:    "journal_max_size_bytes_wrong_type",
			key:     "journal_max_size_bytes",
			value:   "not_a_number",
			errText: "expected int64",
		},
		{
			name:    "memory_limit_bytes_wrong_type",
			key:     "memory_limit_bytes",
			value:   "not_a_number",
			errText: "expected int64",
		},
		{
			name:    "memory_degrade_threshold_percent_wrong_type",
			key:     "memory_degrade_threshold_percent",
			value:   "not_a_number",
			errText: "expected int",
		},
		{
			name:    "risk_max_position_wrong_type",
			key:     "risk.max_position",
			value:   123,
			errText: "expected string",
		},
		{
			name:    "exchange_api_key_id_wrong_type",
			key:     "exchange.api_key_id",
			value:   123,
			errText: "expected string",
		},
		{
			name:    "exchange_default_symbol_wrong_type",
			key:     "exchange.default_symbol",
			value:   123,
			errText: "expected string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := map[string]interface{}{tt.key: tt.value}
			err := repo.UpdateConfig(ctx, updates)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errText)
		})
	}
}

func TestRepository_UpdateConfig_NoCurrentConfig(t *testing.T) {
	repo := &Repository{manager: &config.Manager{}}
	ctx := context.Background()

	updates := map[string]interface{}{"export_timezone": "UTC"}
	err := repo.UpdateConfig(ctx, updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no current configuration")
}
