package agent

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

type runtimeRepository interface {
	Status(context.Context) (entity.RuntimeStatus, error)
	Transition(context.Context, entity.RunMode, string) error
	SetLicense(context.Context, entity.LicenseState) error
	SetConnectivity(context.Context, entity.ConnectivityState) error
}

type configRepository interface {
	Config(context.Context) (entity.AgentConfig, error)
	SetConfig(context.Context, entity.AgentConfig) error
	UpdateConfig(context.Context, map[string]interface{}) error
}

type disclaimerRepository interface {
	DisclaimerAccepted(context.Context) (bool, error)
	AcceptDisclaimer(context.Context, string) error
}

type exportRepository interface {
	ExportCSV(context.Context, ExportCSVInput) (ExportCSVOutput, error)
}

type pluginTrustRepository interface {
	TrustPluginKey(context.Context, TrustPluginKeyInput) error
}

// Service describes the agent application use cases exposed to presentation layers.
type Service interface {
	AcceptDisclaimer(context.Context, string) error
	Check(context.Context) (CheckOutput, error)
	Status(context.Context) (entity.RuntimeStatus, error)
	Config(context.Context) (entity.AgentConfig, error)
	StartScan(context.Context) error
	StartPaper(context.Context) error
	StartLive(context.Context) error
	QueueBacktest(context.Context) error
	StopAll(context.Context) error
	EmergencyStop(context.Context) error
	ExportCSV(context.Context, ExportCSVInput) (ExportCSVOutput, error)
	TrustPluginKey(context.Context, TrustPluginKeyInput) error
	SetConfig(context.Context, SetConfigInput) error
	UpdateConfig(context.Context, UpdateConfigInput) error
	Entitlement(context.Context) (EntitlementOutput, error)
	CanUseFeature(context.Context, string) (bool, error)
}

type EntitlementOutput struct {
	Edition  string
	State    string
	Limits   entity.Limits
	Features []string
}
