package config

import "time"

// TelemetryProfile controls outbound telemetry from the local config layer.
type TelemetryProfile string

const (
	TelemetryProfileOff     TelemetryProfile = "off"
	TelemetryProfileMinimal TelemetryProfile = "minimal"
	TelemetryProfileSupport TelemetryProfile = "support"
	TelemetryProfileSync    TelemetryProfile = "sync"
)

// Valid reports whether the telemetry profile is recognized.
func (p TelemetryProfile) Valid() bool {
	switch p {
	case TelemetryProfileOff, TelemetryProfileMinimal, TelemetryProfileSupport, TelemetryProfileSync:
		return true
	default:
		return false
	}
}

// TelemetryConsent records the operator's explicit telemetry consent.
type TelemetryConsent struct {
	Enabled            bool             `yaml:"enabled"`
	Profile            TelemetryProfile `yaml:"profile"`
	ConsentVersion     string           `yaml:"consent_version"`
	ConsentSource      string           `yaml:"consent_source"`
	ConsentedAtUTC     time.Time        `yaml:"consented_at_utc"`
	AnonymousInstallID string           `yaml:"anonymous_install_id"`
}
