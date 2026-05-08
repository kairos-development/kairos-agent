package agent

import "github.com/kairos-development/kairos-agent/internal/domain/entity"

// CheckOutput is the service-layer status report returned by kairos check.
type CheckOutput struct {
	SchemaVersion   int
	Telemetry       entity.TelemetryConsent
	AllowedOutbound []string
	Status          entity.RuntimeStatus
}

// ExportCSVInput requests a CSV export to the provided path.
type ExportCSVInput struct {
	DestinationPath string
}

// ExportCSVOutput contains the created export path.
type ExportCSVOutput struct {
	Path string
}

// TrustPluginKeyInput requests trust for a plugin signing key.
type TrustPluginKeyInput struct {
	Fingerprint string
	PublicKey   []byte
}

// SetConfigInput sets a complete configuration.
type SetConfigInput struct {
	Config entity.AgentConfig
}

// UpdateConfigInput updates specific configuration fields.
type UpdateConfigInput struct {
	Key   string
	Value interface{}
}
