package tui

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
)

type agentService interface {
	Status(context.Context) (entity.RuntimeStatus, error)
	StartScan(context.Context) error
	StartPaper(context.Context) error
	StartLive(context.Context) error
	QueueBacktest(context.Context) error
	StopAll(context.Context) error
	ExportCSV(context.Context, serviceagent.ExportCSVInput) (serviceagent.ExportCSVOutput, error)
	SetConfig(context.Context, serviceagent.SetConfigInput) error
	UpdateConfig(context.Context, serviceagent.UpdateConfigInput) error
}
