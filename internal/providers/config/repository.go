package configprovider

import (
	"context"
	"fmt"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	providerconv "github.com/kairos-development/kairos-agent/internal/providers/conv"
)

// Repository adapts config storage to service-layer interfaces.
type Repository struct {
	manager *config.Manager
}

// New creates a config provider repository.
func New(manager *config.Manager) *Repository {
	return &Repository{manager: manager}
}

// Config returns the current domain configuration.
func (r *Repository) Config(context.Context) (entity.AgentConfig, error) {
	return providerconv.ConfigToDomain(r.manager.Current()), nil
}

// SetConfig replaces the entire configuration.
func (r *Repository) SetConfig(ctx context.Context, cfg entity.AgentConfig) error {
	infraConfig := providerconv.ConfigFromDomain(cfg)
	return r.manager.Apply(infraConfig)
}

// UpdateConfig updates specific configuration fields.
func (r *Repository) UpdateConfig(ctx context.Context, updates map[string]interface{}) error {
	current := r.manager.Current()
	if current == nil {
		return fmt.Errorf("no current configuration")
	}

	for key, value := range updates {
		if err := applyConfigUpdate(current, key, value); err != nil {
			return fmt.Errorf("update %s: %w", key, err)
		}
	}

	return r.manager.Apply(current)
}

func applyConfigUpdate(cfg *config.Config, key string, value interface{}) error {
	switch key {
	case "export_timezone":
		if v, ok := value.(string); ok {
			cfg.ExportTimezone = v
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "telemetry.enabled":
		if v, ok := value.(bool); ok {
			cfg.Telemetry.Enabled = v
		} else {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case "telemetry.profile":
		if v, ok := value.(string); ok {
			cfg.Telemetry.Profile = config.TelemetryProfile(v)
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "journal_max_size_bytes":
		if v, ok := value.(int64); ok {
			cfg.JournalMaxSizeBytes = v
		} else if v, ok := value.(int); ok {
			cfg.JournalMaxSizeBytes = int64(v)
		} else {
			return fmt.Errorf("expected int64, got %T", value)
		}
	case "memory_limit_bytes":
		if v, ok := value.(int64); ok {
			cfg.MemoryLimitBytes = v
		} else if v, ok := value.(int); ok {
			cfg.MemoryLimitBytes = int64(v)
		} else {
			return fmt.Errorf("expected int64, got %T", value)
		}
	case "memory_degrade_threshold_percent":
		if v, ok := value.(int); ok {
			cfg.MemoryDegradeThresholdPct = v
		} else {
			return fmt.Errorf("expected int, got %T", value)
		}
	case "risk.max_position":
		if v, ok := value.(string); ok {
			cfg.Risk.MaxPosition = v
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "exchange.api_key_id":
		if v, ok := value.(string); ok {
			cfg.Exchange.APIKeyID = v
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "exchange.default_symbol":
		if v, ok := value.(string); ok {
			cfg.Exchange.DefaultSymbol = v
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}
