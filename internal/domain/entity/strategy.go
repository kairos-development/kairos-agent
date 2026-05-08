package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// StrategyStatus identifies the strategy lifecycle state.
type StrategyStatus string

const (
	StrategyStatusIdle    StrategyStatus = "idle"
	StrategyStatusActive  StrategyStatus = "active"
	StrategyStatusPaused  StrategyStatus = "paused"
	StrategyStatusStopped StrategyStatus = "stopped"
)

// StrategyType identifies the strategy execution mode.
type StrategyType string

const (
	StrategyTypeBacktest StrategyType = "backtest"
	StrategyTypePaper    StrategyType = "paper"
	StrategyTypeLive     StrategyType = "live"
)

// Strategy represents a trading strategy in the domain layer.
type Strategy struct {
	ID              string
	Name            string
	Type            StrategyType
	Status          StrategyStatus
	Symbol          string
	MaxPositionSize decimal.Decimal
	MaxDailyLoss    decimal.Decimal
	CurrentPnL      decimal.Decimal
	DailyPnL        decimal.Decimal
	TotalTrades     int
	WinningTrades   int
	LosingTrades    int
	CreatedAtUTC    time.Time
	UpdatedAtUTC    time.Time
	StartedAtUTC    *time.Time
	StoppedAtUTC    *time.Time
}

// IsActive returns true if the strategy is currently running.
func (s *Strategy) IsActive() bool {
	return s.Status == StrategyStatusActive
}

// IsPaused returns true if the strategy is paused.
func (s *Strategy) IsPaused() bool {
	return s.Status == StrategyStatusPaused
}

// IsStopped returns true if the strategy has been stopped.
func (s *Strategy) IsStopped() bool {
	return s.Status == StrategyStatusStopped
}

// CanTrade returns true if the strategy can execute new trades.
func (s *Strategy) CanTrade() bool {
	return s.Status == StrategyStatusActive
}

// HasReachedDailyLossLimit returns true if daily loss limit is breached.
func (s *Strategy) HasReachedDailyLossLimit() bool {
	if s.MaxDailyLoss.IsZero() {
		return false
	}
	return s.DailyPnL.LessThanOrEqual(s.MaxDailyLoss.Neg())
}

// WinRate returns the percentage of winning trades.
func (s *Strategy) WinRate() decimal.Decimal {
	if s.TotalTrades == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(int64(s.WinningTrades)).
		Div(decimal.NewFromInt(int64(s.TotalTrades))).
		Mul(decimal.NewFromInt(100))
}

// UpdatePnL updates the strategy PnL metrics.
func (s *Strategy) UpdatePnL(tradePnL decimal.Decimal) {
	s.CurrentPnL = s.CurrentPnL.Add(tradePnL)
	s.DailyPnL = s.DailyPnL.Add(tradePnL)
	s.TotalTrades++

	if tradePnL.GreaterThan(decimal.Zero) {
		s.WinningTrades++
	} else if tradePnL.LessThan(decimal.Zero) {
		s.LosingTrades++
	}

	s.UpdatedAtUTC = time.Now().UTC()
}

// ResetDailyPnL resets the daily PnL counter.
// This should be called at the start of each trading day.
func (s *Strategy) ResetDailyPnL() {
	s.DailyPnL = decimal.Zero
	s.UpdatedAtUTC = time.Now().UTC()
}
