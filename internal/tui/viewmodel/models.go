package viewmodel

import (
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// Dashboard - Dashboard view model with aggregated metrics.
type Dashboard struct {
	SystemStatus     SystemStatus
	Performance      Performance
	ActiveStrategies []Strategy
	ActiveOrders     []Order // Short list of active orders (5-10 max)
	RecentActivity   []Activity
}

// SystemStatus - System status information.
type SystemStatus struct {
	Mode         string
	Connectivity string
	License      string
	Uptime       string
	NTPDrift     string
}

// Performance - Performance metrics.
type Performance struct {
	TotalPnL      string
	DailyPnL      string
	OpenPositions int
	ActiveOrders  int
	WinRate       string
	TotalTrades   int
}

// Strategy - Strategy information.
type Strategy struct {
	Name   string
	Status string
	PnL    string
}

// Order - Order information for dashboard.
type Order struct {
	Symbol string
	Side   string
	Type   string
	Qty    string
	Price  string
	Status string
}

// Activity - Recent activity entry.
type Activity struct {
	Timestamp string
	Message   string
}

// Positions - Positions view model.
type Positions struct {
	Positions []Position
	TotalPnL  string
}

// Position - Position information.
type Position struct {
	Symbol       string
	Side         string
	Qty          string
	EntryPrice   string
	CurrentPrice string
	PnL          string
	PnLPercent   string
	EntryTime    string
	StrategyID   string
}

// Balance - Balance view model.
type Balance struct {
	Balances    []BalanceItem
	TotalEquity string
}

// BalanceItem - Balance item information.
type BalanceItem struct {
	Asset     string
	Total     string
	Available string
	Locked    string
}

// Market - Market ticker view model.
type Market struct {
	Symbol    string
	LastPrice string
	BidPrice  string
	AskPrice  string
	Spread    string
	SpreadPct string
	Volume24h string
}

// Settings - Settings view model (read-only).
type Settings struct {
	Risk    []ConfigItem
	Trading []ConfigItem
	System  []ConfigItem
}

// ConfigItem - Configuration item.
type ConfigItem struct {
	Key   string
	Value string
}

// NewDashboard creates a dashboard view model from domain entities.
func NewDashboard(status entity.RuntimeStatus, positions []*entity.Position, orders []*entity.Order, strategies []*entity.Strategy) *Dashboard {
	// Calculate total PnL from positions
	totalPnL := decimal.Zero
	dailyPnL := decimal.Zero
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	for _, pos := range positions {
		totalPnL = totalPnL.Add(pos.UnrealizedPnL)
		if pos.OpenedAtUTC.After(dayStart) {
			dailyPnL = dailyPnL.Add(pos.UnrealizedPnL)
		}
	}

	// Limit orders to 5 most recent
	maxOrders := 5
	if len(orders) > maxOrders {
		orders = orders[:maxOrders]
	}

	// Transform orders
	orderVMs := make([]Order, len(orders))
	for i, order := range orders {
		orderVMs[i] = Order{
			Symbol: order.Symbol,
			Side:   string(order.Side),
			Type:   string(order.Type),
			Qty:    order.Quantity.String(),
			Price:  order.Price.String(),
			Status: string(order.Status),
		}
	}

	// Transform strategies
	strategyVMs := make([]Strategy, len(strategies))
	for i, strat := range strategies {
		// Calculate PnL for this strategy
		stratPnL := decimal.Zero
		for _, pos := range positions {
			if pos.StrategyID == strat.ID {
				stratPnL = stratPnL.Add(pos.UnrealizedPnL)
			}
		}

		strategyVMs[i] = Strategy{
			Name:   strat.Name,
			Status: string(strat.Status),
			PnL:    formatMoney(stratPnL),
		}
	}

	return &Dashboard{
		SystemStatus: SystemStatus{
			Mode:         string(status.Mode),
			Connectivity: string(status.Connectivity),
			License:      string(status.License),
			Uptime:       formatDuration(time.Since(status.LastUpdatedAtUTC)),
			NTPDrift:     formatDuration(status.NTPDrift),
		},
		Performance: Performance{
			TotalPnL:      formatMoney(totalPnL),
			DailyPnL:      formatMoney(dailyPnL),
			OpenPositions: len(positions),
			ActiveOrders:  len(orders),
			WinRate:       "0.0%", // TODO: Calculate win rate from trade history
			TotalTrades:   0,      // TODO: Get from trades repository
		},
		ActiveStrategies: strategyVMs,
		ActiveOrders:     orderVMs,
		RecentActivity:   []Activity{}, // TODO: Get from event log
	}
}

// NewPositions creates a positions view model from domain entities.
func NewPositions(positions []*entity.Position) *Positions {
	totalPnL := decimal.Zero
	posVMs := make([]Position, len(positions))

	for i, pos := range positions {
		totalPnL = totalPnL.Add(pos.UnrealizedPnL)

		pnlPercent := decimal.Zero
		if !pos.EntryPrice.IsZero() {
			pnlPercent = pos.UnrealizedPnL.Div(pos.EntryPrice.Mul(pos.Quantity)).Mul(decimal.NewFromInt(100))
		}

		posVMs[i] = Position{
			Symbol:       pos.Symbol,
			Side:         string(pos.Side),
			Qty:          pos.Quantity.String(),
			EntryPrice:   pos.EntryPrice.String(),
			CurrentPrice: pos.CurrentPrice.String(),
			PnL:          formatMoney(pos.UnrealizedPnL),
			PnLPercent:   fmt.Sprintf("%.2f%%", pnlPercent.InexactFloat64()),
			EntryTime:    pos.OpenedAtUTC.Format("15:04:05"),
			StrategyID:   pos.StrategyID,
		}
	}

	return &Positions{
		Positions: posVMs,
		TotalPnL:  formatMoney(totalPnL),
	}
}

// NewBalance creates a balance view model from domain entity.
func NewBalance(balance *entity.AccountBalance) *Balance {
	balVMs := make([]BalanceItem, len(balance.Balances))

	for i, bal := range balance.Balances {
		balVMs[i] = BalanceItem{
			Asset:     bal.Asset,
			Total:     bal.Total.String(),
			Available: bal.Available.String(),
			Locked:    bal.Locked.String(),
		}
	}

	// TODO: Calculate total equity in USD
	totalEquity := "$0.00"

	return &Balance{
		Balances:    balVMs,
		TotalEquity: totalEquity,
	}
}

// NewMarket creates a market view model.
func NewMarket(symbol string) *Market {
	// TODO: Fetch real market data from connector
	return &Market{
		Symbol:    symbol,
		LastPrice: "0.00",
		BidPrice:  "0.00",
		AskPrice:  "0.00",
		Spread:    "$0.00",
		SpreadPct: "0.00%",
		Volume24h: "0.00",
	}
}

// NewSettings creates a settings view model from domain entity.
func NewSettings(cfg entity.AgentConfig) *Settings {
	return &Settings{
		Risk: []ConfigItem{
			{Key: "max_position_size", Value: cfg.Risk.MaxPosition},
		},
		Trading: []ConfigItem{
			{Key: "default_symbol", Value: cfg.Exchange.DefaultSymbol},
		},
		System: []ConfigItem{
			{Key: "telemetry_enabled", Value: fmt.Sprintf("%v", cfg.Telemetry.Enabled)},
			{Key: "telemetry_profile", Value: string(cfg.Telemetry.Profile)},
			{Key: "export_timezone", Value: cfg.ExportTimezone},
			{Key: "journal_max_size_mb", Value: fmt.Sprintf("%d", cfg.JournalMaxSizeBytes/(1024*1024))},
		},
	}
}

// formatMoney formats a decimal as money string.
func formatMoney(d decimal.Decimal) string {
	if d.IsNegative() {
		return fmt.Sprintf("-$%s", d.Abs().StringFixed(2))
	}
	return fmt.Sprintf("+$%s", d.StringFixed(2))
}

// formatDuration formats a duration as human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
