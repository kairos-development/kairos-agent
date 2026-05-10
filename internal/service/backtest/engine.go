package backtest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// HistoricalDataProvider provides historical market data for backtesting.
type HistoricalDataProvider interface {
	GetCandles(ctx context.Context, symbol string, interval string, start, end time.Time) ([]*Candle, error)
}

// Strategy defines the interface for backtest strategies.
type Strategy interface {
	OnTick(ctx context.Context, tick *Tick) (*Signal, error)
	OnCandle(ctx context.Context, candle *Candle) (*Signal, error)
}

// Candle represents a historical price candle.
type Candle struct {
	Symbol    string
	Timestamp time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
}

// Tick represents a price tick.
type Tick struct {
	Symbol    string
	Timestamp time.Time
	Price     decimal.Decimal
	Volume    decimal.Decimal
}

// Signal represents a trading signal from a strategy.
type Signal struct {
	Action   SignalAction
	Symbol   string
	Quantity decimal.Decimal
	Price    decimal.Decimal
}

// SignalAction defines the type of trading signal.
type SignalAction string

const (
	SignalActionBuy  SignalAction = "buy"
	SignalActionSell SignalAction = "sell"
	SignalActionHold SignalAction = "hold"
)

// Engine executes deterministic backtests on historical data.
type Engine struct {
	mu sync.Mutex

	logger *logrus.Logger

	// Backtest state
	currentTime    time.Time
	initialBalance decimal.Decimal
	balance        decimal.Decimal
	positions      map[string]*Position
	trades         []*Trade
	equity         []EquityPoint

	// Configuration
	commissionRate decimal.Decimal
	slippageBps    decimal.Decimal

	// Deterministic RNG seed
	seed int64
}

// Position represents a backtest position.
type Position struct {
	Symbol     string
	Quantity   decimal.Decimal
	EntryPrice decimal.Decimal
	EntryTime  time.Time
}

// Trade represents a completed backtest trade.
type Trade struct {
	Symbol     string
	Side       entity.OrderSide
	Quantity   decimal.Decimal
	EntryPrice decimal.Decimal
	ExitPrice  decimal.Decimal
	EntryTime  time.Time
	ExitTime   time.Time
	PnL        decimal.Decimal
	Commission decimal.Decimal
}

// EquityPoint represents equity at a point in time.
type EquityPoint struct {
	Timestamp time.Time
	Equity    decimal.Decimal
}

// Config contains backtest engine configuration.
type Config struct {
	InitialBalance decimal.Decimal
	CommissionRate decimal.Decimal
	SlippageBps    decimal.Decimal
	Seed           int64
}

// DefaultConfig returns default backtest configuration.
func DefaultConfig() *Config {
	return &Config{
		InitialBalance: decimal.NewFromInt(10000),
		CommissionRate: decimal.NewFromInt(6).Shift(-4), // 0.06%
		SlippageBps:    decimal.NewFromInt(5),           // 0.05%
		Seed:           time.Now().UnixNano(),
	}
}

// NewEngine creates a new backtest engine.
func NewEngine(config *Config, logger *logrus.Logger) *Engine {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	return &Engine{
		logger:         logger,
		initialBalance: config.InitialBalance,
		balance:        config.InitialBalance,
		positions:      make(map[string]*Position),
		trades:         make([]*Trade, 0),
		equity:         make([]EquityPoint, 0),
		commissionRate: config.CommissionRate,
		slippageBps:    config.SlippageBps,
		seed:           config.Seed,
	}
}

// Run executes a backtest with the given strategy and historical data.
func (e *Engine) Run(ctx context.Context, strategy Strategy, candles []*Candle) (*Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.WithFields(logrus.Fields{
		"candles":         len(candles),
		"initial_balance": e.initialBalance.String(),
		"seed":            e.seed,
	}).Info("Starting backtest")

	startTime := time.Now()

	// Process each candle deterministically
	for i, candle := range candles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		e.currentTime = candle.Timestamp

		// Get strategy signal
		signal, err := strategy.OnCandle(ctx, candle)
		if err != nil {
			e.logger.WithError(err).Error("Strategy error")
			continue
		}

		if signal == nil || signal.Action == SignalActionHold {
			continue
		}

		// Execute signal
		if err := e.executeSignal(signal, candle.Close); err != nil {
			e.logger.WithError(err).Warn("Failed to execute signal")
			continue
		}

		// Record equity point every 100 candles
		if i%100 == 0 {
			e.recordEquity()
		}
	}

	// Close all open positions at end
	e.closeAllPositions(candles[len(candles)-1].Close)

	// Final equity point
	e.recordEquity()

	duration := time.Since(startTime)

	result := e.calculateResult(duration)

	e.logger.WithFields(logrus.Fields{
		"total_trades":   result.TotalTrades,
		"winning_trades": result.WinningTrades,
		"final_equity":   result.FinalEquity.String(),
		"total_return":   result.TotalReturn.String(),
		"duration_ms":    duration.Milliseconds(),
	}).Info("Backtest completed")

	return result, nil
}

// Reset resets the backtest engine state.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.balance = e.initialBalance
	e.positions = make(map[string]*Position)
	e.trades = make([]*Trade, 0)
	e.equity = make([]EquityPoint, 0)
	e.currentTime = time.Time{}
}

func (e *Engine) executeSignal(signal *Signal, currentPrice decimal.Decimal) error {
	switch signal.Action {
	case SignalActionBuy:
		return e.openPosition(signal.Symbol, signal.Quantity, currentPrice)
	case SignalActionSell:
		return e.closePosition(signal.Symbol, currentPrice)
	default:
		return nil
	}
}

func (e *Engine) openPosition(symbol string, quantity decimal.Decimal, price decimal.Decimal) error {
	// Check if position already exists
	if _, exists := e.positions[symbol]; exists {
		return fmt.Errorf("position already open for %s", symbol)
	}

	// Apply slippage
	fillPrice := e.applySlippage(price, true)

	// Calculate cost
	notional := quantity.Mul(fillPrice)
	commission := notional.Mul(e.commissionRate)
	totalCost := notional.Add(commission)

	// Check balance
	if e.balance.LessThan(totalCost) {
		return fmt.Errorf("insufficient balance: have %s, need %s", e.balance.String(), totalCost.String())
	}

	// Deduct from balance
	e.balance = e.balance.Sub(totalCost)

	// Create position
	e.positions[symbol] = &Position{
		Symbol:     symbol,
		Quantity:   quantity,
		EntryPrice: fillPrice,
		EntryTime:  e.currentTime,
	}

	e.logger.WithFields(logrus.Fields{
		"symbol":     symbol,
		"quantity":   quantity.String(),
		"fill_price": fillPrice.String(),
		"commission": commission.String(),
	}).Debug("Position opened")

	return nil
}

func (e *Engine) closePosition(symbol string, price decimal.Decimal) error {
	position, exists := e.positions[symbol]
	if !exists {
		return fmt.Errorf("no position to close for %s", symbol)
	}

	// Apply slippage
	fillPrice := e.applySlippage(price, false)

	// Calculate proceeds
	notional := position.Quantity.Mul(fillPrice)
	commission := notional.Mul(e.commissionRate)
	proceeds := notional.Sub(commission)

	// Add to balance
	e.balance = e.balance.Add(proceeds)

	// Calculate PnL
	pnl := fillPrice.Sub(position.EntryPrice).Mul(position.Quantity).Sub(commission)

	// Record trade
	trade := &Trade{
		Symbol:     symbol,
		Side:       entity.OrderSideBuy,
		Quantity:   position.Quantity,
		EntryPrice: position.EntryPrice,
		ExitPrice:  fillPrice,
		EntryTime:  position.EntryTime,
		ExitTime:   e.currentTime,
		PnL:        pnl,
		Commission: commission,
	}
	e.trades = append(e.trades, trade)

	// Remove position
	delete(e.positions, symbol)

	e.logger.WithFields(logrus.Fields{
		"symbol":     symbol,
		"pnl":        pnl.String(),
		"fill_price": fillPrice.String(),
	}).Debug("Position closed")

	return nil
}

func (e *Engine) closeAllPositions(currentPrice decimal.Decimal) {
	for symbol := range e.positions {
		if err := e.closePosition(symbol, currentPrice); err != nil {
			e.logger.WithError(err).Warn("Failed to close position")
		}
	}
}

func (e *Engine) applySlippage(price decimal.Decimal, isBuy bool) decimal.Decimal {
	slippageMultiplier := e.slippageBps.Div(decimal.NewFromInt(10000))

	if isBuy {
		return price.Mul(decimal.NewFromInt(1).Add(slippageMultiplier))
	}
	return price.Mul(decimal.NewFromInt(1).Sub(slippageMultiplier))
}

func (e *Engine) recordEquity() {
	equity := e.balance

	// Add unrealized PnL from open positions
	for _, position := range e.positions {
		// Use entry price as current price approximation
		// In real backtest, we'd use the current market price
		equity = equity.Add(position.Quantity.Mul(position.EntryPrice))
	}

	e.equity = append(e.equity, EquityPoint{
		Timestamp: e.currentTime,
		Equity:    equity,
	})
}

func (e *Engine) calculateResult(duration time.Duration) *Result {
	finalEquity := e.balance

	// Add value of open positions (shouldn't be any after closeAll)
	for _, position := range e.positions {
		finalEquity = finalEquity.Add(position.Quantity.Mul(position.EntryPrice))
	}

	totalReturn := finalEquity.Sub(e.initialBalance).Div(e.initialBalance).Mul(decimal.NewFromInt(100))

	winningTrades := 0
	losingTrades := 0
	totalProfit := decimal.Zero
	totalLoss := decimal.Zero

	for _, trade := range e.trades {
		if trade.PnL.GreaterThan(decimal.Zero) {
			winningTrades++
			totalProfit = totalProfit.Add(trade.PnL)
		} else {
			losingTrades++
			totalLoss = totalLoss.Add(trade.PnL.Abs())
		}
	}

	winRate := decimal.Zero
	if len(e.trades) > 0 {
		winRate = decimal.NewFromInt(int64(winningTrades)).Div(decimal.NewFromInt(int64(len(e.trades)))).Mul(decimal.NewFromInt(100))
	}

	profitFactor := decimal.Zero
	if !totalLoss.IsZero() {
		profitFactor = totalProfit.Div(totalLoss)
	}

	return &Result{
		InitialBalance: e.initialBalance,
		FinalEquity:    finalEquity,
		TotalReturn:    totalReturn,
		TotalTrades:    len(e.trades),
		WinningTrades:  winningTrades,
		LosingTrades:   losingTrades,
		WinRate:        winRate,
		TotalProfit:    totalProfit,
		TotalLoss:      totalLoss,
		ProfitFactor:   profitFactor,
		Equity:         e.equity,
		Trades:         e.trades,
		Duration:       duration,
		Seed:           e.seed,
	}
}

// Result contains backtest results.
type Result struct {
	InitialBalance decimal.Decimal
	FinalEquity    decimal.Decimal
	TotalReturn    decimal.Decimal
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	WinRate        decimal.Decimal
	TotalProfit    decimal.Decimal
	TotalLoss      decimal.Decimal
	ProfitFactor   decimal.Decimal
	Equity         []EquityPoint
	Trades         []*Trade
	Duration       time.Duration
	Seed           int64
}
