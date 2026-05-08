package actions

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/service/agent"
)

// ActionService provides command/mutation operations for TUI.
// This is a TUI-owned adapter interface, NOT a business service.
// Implementation delegates to existing application/service layer.
type ActionService interface {
	// StartPaper switches to paper trading mode
	StartPaper(ctx context.Context) error

	// StartLive switches to live trading mode
	StartLive(ctx context.Context) error

	// StartScan switches to scanning mode
	StartScan(ctx context.Context) error

	// StopAll stops all trading activity
	StopAll(ctx context.Context) error

	// QueueBacktest queues a backtest
	QueueBacktest(ctx context.Context) error

	// UpdateConfig updates configuration
	UpdateConfig(ctx context.Context, key string, value interface{}) error

	// ExportCSV exports data to CSV
	ExportCSV(ctx context.Context, input agent.ExportCSVInput) (agent.ExportCSVOutput, error)
}

// actionService implements ActionService by delegating to application service layer.
type actionService struct {
	agentService agent.Service
}

// NewActionService creates a new action service.
func NewActionService(agentService agent.Service) ActionService {
	return &actionService{
		agentService: agentService,
	}
}

// StartPaper switches to paper trading mode.
func (s *actionService) StartPaper(ctx context.Context) error {
	return s.agentService.StartPaper(ctx)
}

// StartLive switches to live trading mode.
func (s *actionService) StartLive(ctx context.Context) error {
	return s.agentService.StartLive(ctx)
}

// StartScan switches to scanning mode.
func (s *actionService) StartScan(ctx context.Context) error {
	return s.agentService.StartScan(ctx)
}

// StopAll stops all trading activity.
func (s *actionService) StopAll(ctx context.Context) error {
	return s.agentService.StopAll(ctx)
}

// QueueBacktest queues a backtest.
func (s *actionService) QueueBacktest(ctx context.Context) error {
	return s.agentService.QueueBacktest(ctx)
}

// UpdateConfig updates configuration.
func (s *actionService) UpdateConfig(ctx context.Context, key string, value interface{}) error {
	input := agent.UpdateConfigInput{
		Key:   key,
		Value: value,
	}
	return s.agentService.UpdateConfig(ctx, input)
}

// ExportCSV exports data to CSV.
func (s *actionService) ExportCSV(ctx context.Context, input agent.ExportCSVInput) (agent.ExportCSVOutput, error) {
	return s.agentService.ExportCSV(ctx, input)
}
