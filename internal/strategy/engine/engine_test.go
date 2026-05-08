package engine

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStrategy struct {
	name      string
	version   string
	signal    *strategy.Signal
	signalErr error
}

func (m *mockStrategy) Name() string    { return m.name }
func (m *mockStrategy) Version() string { return m.version }
func (m *mockStrategy) OnTick(ctx context.Context, data *strategy.StrategyContext) (*strategy.Signal, error) {
	return m.signal, m.signalErr
}
func (m *mockStrategy) OnOrderFilled(ctx context.Context, orderID string, filledQty, fillPrice decimal.Decimal) error {
	return nil
}
func (m *mockStrategy) OnOrderCanceled(ctx context.Context, orderID string) error { return nil }
func (m *mockStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{"test": true}
}
func (m *mockStrategy) SetParameters(params map[string]interface{}) error { return nil }
func (m *mockStrategy) Reset() error                                      { return nil }

func TestEngine_Tick_GeneratesSignal(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "test-strategy",
		version: "1.0.0",
		signal: &strategy.Signal{
			StrategyID:  "test-1",
			Symbol:      "BTCUSDT",
			Action:      strategy.SignalActionBuy,
			Quantity:    decimal.NewFromInt(1),
			Price:       decimal.NewFromInt(50000),
			Confidence:  decimal.NewFromFloat(0.95),
			Reason:      "test signal",
			GeneratedAt: time.Now().UTC(),
		},
	}
	require.NoError(t, registry.Register("test-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	signal, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-1",
		StrategyID: "test-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})

	require.NoError(t, err)
	require.NotNil(t, signal)
	assert.Equal(t, strategy.SignalActionBuy, signal.Action)
	assert.True(t, signal.Quantity.Equal(decimal.NewFromInt(1)))
}

func TestEngine_Tick_NoSignal(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "hold-strategy",
		version: "1.0.0",
		signal:  nil,
	}
	require.NoError(t, registry.Register("hold-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	signal, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-2",
		StrategyID: "hold-1",
		Symbol:     "ETHUSDT",
		LastPrice:  decimal.NewFromInt(3000),
		Timestamp:  time.Now().UTC(),
	})

	require.NoError(t, err)
	assert.Nil(t, signal)
}

func TestEngine_OnOrderFilled_UpdatesState(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "fill-strategy",
		version: "1.0.0",
	}
	require.NoError(t, registry.Register("fill-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-3",
		StrategyID: "fill-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	newPos := strategy.PositionInfo{
		Symbol:        "BTCUSDT",
		Side:          "long",
		Quantity:      decimal.NewFromInt(1),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(50500),
		UnrealizedPnL: decimal.NewFromInt(500),
	}
	newBal := strategy.BalanceInfo{
		Total:     decimal.NewFromInt(10000),
		Available: decimal.NewFromInt(5000),
		Locked:    decimal.NewFromInt(5000),
	}

	err = engine.OnOrderFilled(ctx, OrderFilledInput{
		InstanceID:  "inst-3",
		OrderID:     "order-1",
		FilledQty:   decimal.NewFromInt(1),
		FillPrice:   decimal.NewFromInt(50000),
		NewPosition: newPos,
		NewBalance:  newBal,
	})
	require.NoError(t, err)
}

func TestEngine_Reset_ClearsContext(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "reset-strategy",
		version: "1.0.0",
	}
	require.NoError(t, registry.Register("reset-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-4",
		StrategyID: "reset-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	err = engine.Reset(ctx, "inst-4")
	require.NoError(t, err)

	_, err = engine.Tick(ctx, TickInput{
		InstanceID: "inst-4",
		StrategyID: "reset-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(51000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)
}
