package query

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
)

// QueryService provides UI-safe data snapshots for TUI views.
// Lives inside internal/tui to avoid service layer depending on TUI view models.
type QueryService interface {
	// GetDashboard returns aggregated dashboard metrics
	GetDashboard(ctx context.Context) (*viewmodel.Dashboard, error)

	// GetPositions returns formatted positions for display
	GetPositions(ctx context.Context) (*viewmodel.Positions, error)

	// GetBalance returns account balance with totals
	GetBalance(ctx context.Context) (*viewmodel.Balance, error)

	// GetMarket returns ticker data for active symbols
	GetMarket(ctx context.Context, symbol string) (*viewmodel.Market, error)

	// GetSettings returns current configuration (read-only)
	GetSettings(ctx context.Context) (*viewmodel.Settings, error)
}

// Reader interfaces for dependency injection.
// These match actual domain/service method signatures.

// RuntimeReader provides runtime status information.
type RuntimeReader interface {
	Status(context.Context) (entity.RuntimeStatus, error)
}

// PositionReader provides position data.
type PositionReader interface {
	ListOpen(context.Context) ([]*entity.Position, error)
}

// BalanceReader provides balance data.
type BalanceReader interface {
	GetLatest(context.Context) (*entity.AccountBalance, error)
}

// OrderReader provides order data.
type OrderReader interface {
	ListActive(context.Context) ([]*entity.Order, error)
}

// ConfigReader provides configuration data.
type ConfigReader interface {
	Config(context.Context) (entity.AgentConfig, error)
}

// StrategyReader provides strategy data.
type StrategyReader interface {
	ListActive(context.Context) ([]*entity.Strategy, error)
}

// queryService implements QueryService.
type queryService struct {
	runtime  RuntimeReader
	position PositionReader
	balance  BalanceReader
	order    OrderReader
	config   ConfigReader
	strategy StrategyReader
}

// NewQueryService creates a new query service.
func NewQueryService(
	runtime RuntimeReader,
	position PositionReader,
	balance BalanceReader,
	order OrderReader,
	config ConfigReader,
	strategy StrategyReader,
) QueryService {
	return &queryService{
		runtime:  runtime,
		position: position,
		balance:  balance,
		order:    order,
		config:   config,
		strategy: strategy,
	}
}

// GetDashboard returns aggregated dashboard metrics.
func (s *queryService) GetDashboard(ctx context.Context) (*viewmodel.Dashboard, error) {
	// Get runtime status
	status, err := s.runtime.Status(ctx)
	if err != nil {
		return nil, err
	}

	// Get positions for performance metrics
	positions, err := s.position.ListOpen(ctx)
	if err != nil {
		return nil, err
	}

	// Get active orders (limit to 5 most recent for dashboard)
	orders, err := s.order.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Get active strategies
	strategies, err := s.strategy.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Transform to view model
	return viewmodel.NewDashboard(status, positions, orders, strategies), nil
}

// GetPositions returns formatted positions for display.
func (s *queryService) GetPositions(ctx context.Context) (*viewmodel.Positions, error) {
	positions, err := s.position.ListOpen(ctx)
	if err != nil {
		return nil, err
	}

	return viewmodel.NewPositions(positions), nil
}

// GetBalance returns account balance with totals.
func (s *queryService) GetBalance(ctx context.Context) (*viewmodel.Balance, error) {
	balance, err := s.balance.GetLatest(ctx)
	if err != nil {
		return nil, err
	}

	return viewmodel.NewBalance(balance), nil
}

// GetMarket returns ticker data for active symbols.
func (s *queryService) GetMarket(ctx context.Context, symbol string) (*viewmodel.Market, error) {
	// TODO: Implement market data fetching from connector
	// For now, return placeholder
	return viewmodel.NewMarket(symbol), nil
}

// GetSettings returns current configuration (read-only).
func (s *queryService) GetSettings(ctx context.Context) (*viewmodel.Settings, error) {
	cfg, err := s.config.Config(ctx)
	if err != nil {
		return nil, err
	}

	return viewmodel.NewSettings(cfg), nil
}
