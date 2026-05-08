package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// Mock strategy for testing
type mockStrategy struct {
	onCandleFunc func(ctx context.Context, candle *Candle) (*Signal, error)
}

func (m *mockStrategy) OnTick(ctx context.Context, tick *Tick) (*Signal, error) {
	return nil, nil
}

func (m *mockStrategy) OnCandle(ctx context.Context, candle *Candle) (*Signal, error) {
	if m.onCandleFunc != nil {
		return m.onCandleFunc(ctx, candle)
	}
	return &Signal{Action: SignalActionHold}, nil
}

func TestNewEngine(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config, nil)

	if engine == nil {
		t.Fatal("expected engine to be created")
	}

	if !engine.initialBalance.Equal(config.InitialBalance) {
		t.Errorf("expected initial balance %s, got %s", config.InitialBalance.String(), engine.initialBalance.String())
	}

	if !engine.balance.Equal(config.InitialBalance) {
		t.Errorf("expected balance %s, got %s", config.InitialBalance.String(), engine.balance.String())
	}

	if engine.seed != config.Seed {
		t.Errorf("expected seed %d, got %d", config.Seed, engine.seed)
	}
}

func TestEngine_Run_NoTrades(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config, nil)

	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			return &Signal{Action: SignalActionHold}, nil
		},
	}

	candles := []*Candle{
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Open:      decimal.NewFromInt(50000),
			High:      decimal.NewFromInt(51000),
			Low:       decimal.NewFromInt(49000),
			Close:     decimal.NewFromInt(50500),
			Volume:    decimal.NewFromInt(100),
		},
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
			Open:      decimal.NewFromInt(50500),
			High:      decimal.NewFromInt(52000),
			Low:       decimal.NewFromInt(50000),
			Close:     decimal.NewFromInt(51000),
			Volume:    decimal.NewFromInt(150),
		},
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalTrades != 0 {
		t.Errorf("expected 0 trades, got %d", result.TotalTrades)
	}

	if !result.FinalEquity.Equal(config.InitialBalance) {
		t.Errorf("expected final equity %s, got %s", config.InitialBalance.String(), result.FinalEquity.String())
	}
}

func TestEngine_Run_BuyAndSell(t *testing.T) {
	config := DefaultConfig()
	config.InitialBalance = decimal.NewFromInt(10000)
	engine := NewEngine(config, nil)

	buyExecuted := false
	sellExecuted := false

	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			if candle.Timestamp.Hour() == 0 && !buyExecuted {
				buyExecuted = true
				return &Signal{
					Action:   SignalActionBuy,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.1),
					Price:    candle.Close,
				}, nil
			}
			if candle.Timestamp.Hour() == 2 && !sellExecuted {
				sellExecuted = true
				return &Signal{
					Action:   SignalActionSell,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.1),
					Price:    candle.Close,
				}, nil
			}
			return &Signal{Action: SignalActionHold}, nil
		},
	}

	candles := []*Candle{
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		},
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50500),
		},
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(51000),
		},
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalTrades != 1 {
		t.Errorf("expected 1 trade, got %d", result.TotalTrades)
	}

	if result.WinningTrades != 1 {
		t.Errorf("expected 1 winning trade, got %d", result.WinningTrades)
	}

	if result.FinalEquity.LessThanOrEqual(config.InitialBalance) {
		t.Errorf("expected profit, got final equity %s", result.FinalEquity.String())
	}
}

func TestEngine_Run_InsufficientBalance(t *testing.T) {
	config := DefaultConfig()
	config.InitialBalance = decimal.NewFromInt(100) // Very small balance
	engine := NewEngine(config, nil)

	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			return &Signal{
				Action:   SignalActionBuy,
				Symbol:   "BTCUSDT",
				Quantity: decimal.NewFromFloat(1.0), // Too large
				Price:    decimal.NewFromInt(50000),
			}, nil
		},
	}

	candles := []*Candle{
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		},
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not execute trade due to insufficient balance
	if result.TotalTrades != 0 {
		t.Errorf("expected 0 trades due to insufficient balance, got %d", result.TotalTrades)
	}
}

func TestEngine_Run_PositionAlreadyOpen(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config, nil)

	callCount := 0
	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			callCount++
			// Try to buy twice
			return &Signal{
				Action:   SignalActionBuy,
				Symbol:   "BTCUSDT",
				Quantity: decimal.NewFromFloat(0.1),
				Price:    candle.Close,
			}, nil
		},
	}

	candles := []*Candle{
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		},
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50500),
		},
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Position should be closed at end, resulting in 1 trade
	if result.TotalTrades != 1 {
		t.Errorf("expected 1 trade (position closed at end), got %d", result.TotalTrades)
	}
}

func TestEngine_Run_ContextCancellation(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config, nil)

	strategy := &mockStrategy{}

	candles := make([]*Candle, 1000)
	for i := range candles {
		candles[i] = &Candle{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, i, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := engine.Run(ctx, strategy, candles)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestEngine_Reset(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config, nil)

	// Add some state
	engine.balance = decimal.NewFromInt(5000)
	engine.positions["BTCUSDT"] = &Position{
		Symbol:   "BTCUSDT",
		Quantity: decimal.NewFromFloat(0.1),
	}
	engine.trades = append(engine.trades, &Trade{
		Symbol: "BTCUSDT",
	})
	engine.equity = append(engine.equity, EquityPoint{
		Timestamp: time.Now(),
		Equity:    decimal.NewFromInt(5000),
	})

	// Reset
	engine.Reset()

	if !engine.balance.Equal(engine.initialBalance) {
		t.Errorf("expected balance to be reset to %s, got %s", engine.initialBalance.String(), engine.balance.String())
	}

	if len(engine.positions) != 0 {
		t.Errorf("expected empty positions, got %d", len(engine.positions))
	}

	if len(engine.trades) != 0 {
		t.Errorf("expected empty trades, got %d", len(engine.trades))
	}

	if len(engine.equity) != 0 {
		t.Errorf("expected empty equity, got %d", len(engine.equity))
	}
}

func TestEngine_Run_MultipleSymbols(t *testing.T) {
	config := DefaultConfig()
	config.InitialBalance = decimal.NewFromInt(20000)
	engine := NewEngine(config, nil)

	btcBought := false
	ethBought := false

	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			if candle.Symbol == "BTCUSDT" && !btcBought {
				btcBought = true
				return &Signal{
					Action:   SignalActionBuy,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.1),
					Price:    candle.Close,
				}, nil
			}
			if candle.Symbol == "ETHUSDT" && !ethBought {
				ethBought = true
				return &Signal{
					Action:   SignalActionBuy,
					Symbol:   "ETHUSDT",
					Quantity: decimal.NewFromFloat(1.0),
					Price:    candle.Close,
				}, nil
			}
			return &Signal{Action: SignalActionHold}, nil
		},
	}

	candles := []*Candle{
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		},
		{
			Symbol:    "ETHUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(3000),
		},
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Both positions should be closed at end
	if result.TotalTrades != 2 {
		t.Errorf("expected 2 trades, got %d", result.TotalTrades)
	}
}

func TestEngine_Run_LossScenario(t *testing.T) {
	config := DefaultConfig()
	config.InitialBalance = decimal.NewFromInt(10000)
	engine := NewEngine(config, nil)

	buyExecuted := false
	sellExecuted := false

	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			if candle.Timestamp.Hour() == 0 && !buyExecuted {
				buyExecuted = true
				return &Signal{
					Action:   SignalActionBuy,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.1),
					Price:    candle.Close,
				}, nil
			}
			if candle.Timestamp.Hour() == 1 && !sellExecuted {
				sellExecuted = true
				return &Signal{
					Action:   SignalActionSell,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.1),
					Price:    candle.Close,
				}, nil
			}
			return &Signal{Action: SignalActionHold}, nil
		},
	}

	candles := []*Candle{
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		},
		{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(48000), // Price dropped
		},
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalTrades != 1 {
		t.Errorf("expected 1 trade, got %d", result.TotalTrades)
	}

	if result.LosingTrades != 1 {
		t.Errorf("expected 1 losing trade, got %d", result.LosingTrades)
	}

	if result.FinalEquity.GreaterThanOrEqual(config.InitialBalance) {
		t.Errorf("expected loss, got final equity %s", result.FinalEquity.String())
	}

	if result.TotalReturn.GreaterThanOrEqual(decimal.Zero) {
		t.Errorf("expected negative return, got %s", result.TotalReturn.String())
	}
}

func TestEngine_Run_EquityRecording(t *testing.T) {
	config := DefaultConfig()
	engine := NewEngine(config, nil)

	strategy := &mockStrategy{}

	// Create 250 candles to trigger equity recording (every 100 candles)
	candles := make([]*Candle, 250)
	for i := range candles {
		candles[i] = &Candle{
			Symbol:    "BTCUSDT",
			Timestamp: time.Date(2026, 1, 1, i, 0, 0, 0, time.UTC),
			Close:     decimal.NewFromInt(50000),
		}
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have equity points recorded (every 100 candles + final)
	// With 250 candles: at 0, 100, 200, and final = 4 points minimum
	if len(result.Equity) < 1 {
		t.Errorf("expected at least 1 equity point, got %d", len(result.Equity))
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.InitialBalance.Equal(decimal.NewFromInt(10000)) {
		t.Errorf("expected initial balance 10000, got %s", config.InitialBalance.String())
	}

	if !config.CommissionRate.Equal(decimal.NewFromFloat(0.0006)) {
		t.Errorf("expected commission rate 0.0006, got %s", config.CommissionRate.String())
	}

	if !config.SlippageBps.Equal(decimal.NewFromInt(5)) {
		t.Errorf("expected slippage 5 bps, got %s", config.SlippageBps.String())
	}

	if config.Seed == 0 {
		t.Error("expected non-zero seed")
	}
}

func TestEngine_Run_WinRate(t *testing.T) {
	config := DefaultConfig()
	config.InitialBalance = decimal.NewFromInt(20000)
	engine := NewEngine(config, nil)

	strategy := &mockStrategy{
		onCandleFunc: func(ctx context.Context, candle *Candle) (*Signal, error) {
			hour := candle.Timestamp.Hour()
			// Buy at hours 0, 2, 4
			if hour%2 == 0 && hour < 6 {
				return &Signal{
					Action:   SignalActionBuy,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.05),
					Price:    candle.Close,
				}, nil
			}
			// Sell at hours 1, 3, 5
			if hour%2 == 1 && hour < 6 {
				return &Signal{
					Action:   SignalActionSell,
					Symbol:   "BTCUSDT",
					Quantity: decimal.NewFromFloat(0.05),
					Price:    candle.Close,
				}, nil
			}
			return &Signal{Action: SignalActionHold}, nil
		},
	}

	candles := []*Candle{
		{Symbol: "BTCUSDT", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Close: decimal.NewFromInt(50000)},
		{Symbol: "BTCUSDT", Timestamp: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), Close: decimal.NewFromInt(51000)}, // Win
		{Symbol: "BTCUSDT", Timestamp: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC), Close: decimal.NewFromInt(51000)},
		{Symbol: "BTCUSDT", Timestamp: time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC), Close: decimal.NewFromInt(50000)}, // Loss
		{Symbol: "BTCUSDT", Timestamp: time.Date(2026, 1, 1, 4, 0, 0, 0, time.UTC), Close: decimal.NewFromInt(50000)},
		{Symbol: "BTCUSDT", Timestamp: time.Date(2026, 1, 1, 5, 0, 0, 0, time.UTC), Close: decimal.NewFromInt(52000)}, // Win
	}

	ctx := context.Background()
	result, err := engine.Run(ctx, strategy, candles)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.TotalTrades != 3 {
		t.Errorf("expected 3 trades, got %d", result.TotalTrades)
	}

	if result.WinningTrades != 2 {
		t.Errorf("expected 2 winning trades, got %d", result.WinningTrades)
	}

	if result.LosingTrades != 1 {
		t.Errorf("expected 1 losing trade, got %d", result.LosingTrades)
	}

	expectedWinRate := decimal.NewFromFloat(66.666666666666666)
	if result.WinRate.Sub(expectedWinRate).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		t.Errorf("expected win rate around 66.67%%, got %s%%", result.WinRate.String())
	}
}
