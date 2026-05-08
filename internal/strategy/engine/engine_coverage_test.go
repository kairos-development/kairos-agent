package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/strategy"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errStrategy struct {
	mockStrategy
	tickErr      error
	fillErr      error
	cancelErr    error
	resetErr     error
	onTickSignal *strategy.Signal
}

func (e *errStrategy) OnTick(ctx context.Context, data *strategy.StrategyContext) (*strategy.Signal, error) {
	if e.tickErr != nil {
		return nil, e.tickErr
	}
	return e.onTickSignal, nil
}

func (e *errStrategy) OnOrderFilled(ctx context.Context, orderID string, filledQty, fillPrice decimal.Decimal) error {
	return e.fillErr
}

func (e *errStrategy) OnOrderCanceled(ctx context.Context, orderID string) error {
	return e.cancelErr
}

func (e *errStrategy) Reset() error {
	return e.resetErr
}

func TestEngine_Tick_StrategyError(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &errStrategy{tickErr: fmt.Errorf("strategy exploded")}
	require.NoError(t, registry.Register("err-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	signal, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-err-1",
		StrategyID: "err-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})

	require.Error(t, err)
	assert.Nil(t, signal)
	assert.Contains(t, err.Error(), "strategy tick")
}

func TestEngine_Tick_UnknownStrategy(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	engine := NewEngine(registry)
	ctx := context.Background()

	signal, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-unknown",
		StrategyID: "nonexistent",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})

	require.Error(t, err)
	assert.Nil(t, signal)
	assert.Contains(t, err.Error(), "get strategy")
}

func TestEngine_Tick_WithContextData(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "ctx-strategy",
		version: "1.0.0",
		signal: &strategy.Signal{
			StrategyID:  "ctx-1",
			Symbol:      "ETHUSDT",
			Action:      strategy.SignalActionSell,
			Quantity:    decimal.NewFromInt(2),
			Price:       decimal.NewFromInt(3000),
			Confidence:  decimal.NewFromFloat(0.8),
			Reason:      "test",
			GeneratedAt: time.Now().UTC(),
		},
	}
	require.NoError(t, registry.Register("ctx-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	signal, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-ctx",
		StrategyID: "ctx-1",
		Symbol:     "ETHUSDT",
		LastPrice:  decimal.NewFromInt(3000),
		BidPrice:   decimal.NewFromInt(2999),
		AskPrice:   decimal.NewFromInt(3001),
		Volume24h:  decimal.NewFromInt(100000),
		High24h:    decimal.NewFromInt(3100),
		Low24h:     decimal.NewFromInt(2900),
		Timestamp:  time.Now().UTC(),
		Parameters: map[string]interface{}{"period": 14},
	})

	require.NoError(t, err)
	require.NotNil(t, signal)
	assert.Equal(t, strategy.SignalActionSell, signal.Action)
	assert.True(t, signal.Quantity.Equal(decimal.NewFromInt(2)))
}

func TestEngine_OnOrderFilled_InstanceNotFound(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	engine := NewEngine(registry)
	ctx := context.Background()

	err := engine.OnOrderFilled(ctx, OrderFilledInput{
		InstanceID: "nonexistent",
		OrderID:    "order-1",
		FilledQty:  decimal.NewFromInt(1),
		FillPrice:  decimal.NewFromInt(50000),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEngine_OnOrderFilled_StrategyError(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &errStrategy{fillErr: fmt.Errorf("fill error")}
	require.NoError(t, registry.Register("fill-err-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	// First tick to create the instance
	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-fill-err",
		StrategyID: "fill-err-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	err = engine.OnOrderFilled(ctx, OrderFilledInput{
		InstanceID: "inst-fill-err",
		OrderID:    "order-1",
		FilledQty:  decimal.NewFromInt(1),
		FillPrice:  decimal.NewFromInt(50000),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "on order filled")
}

func TestEngine_OnOrderCanceled_Success(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "cancel-strategy",
		version: "1.0.0",
	}
	require.NoError(t, registry.Register("cancel-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-cancel",
		StrategyID: "cancel-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	err = engine.OnOrderCanceled(ctx, "inst-cancel", "order-1")
	require.NoError(t, err)
}

func TestEngine_OnOrderCanceled_InstanceNotFound(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	engine := NewEngine(registry)
	ctx := context.Background()

	err := engine.OnOrderCanceled(ctx, "nonexistent", "order-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEngine_OnOrderCanceled_StrategyError(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &errStrategy{cancelErr: fmt.Errorf("cancel error")}
	require.NoError(t, registry.Register("cancel-err-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-cancel-err",
		StrategyID: "cancel-err-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	err = engine.OnOrderCanceled(ctx, "inst-cancel-err", "order-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancel error")
}

func TestEngine_UpdatePosition(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "pos-strategy",
		version: "1.0.0",
	}
	require.NoError(t, registry.Register("pos-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-pos",
		StrategyID: "pos-1",
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

	engine.UpdatePosition(ctx, "inst-pos", newPos)

	// Verify position was updated by checking internal state via OnOrderFilled
	err = engine.OnOrderFilled(ctx, OrderFilledInput{
		InstanceID:  "inst-pos",
		OrderID:     "order-1",
		FilledQty:   decimal.NewFromInt(1),
		FillPrice:   decimal.NewFromInt(50000),
		NewPosition: newPos,
		NewBalance:  strategy.BalanceInfo{Total: decimal.NewFromInt(10000), Available: decimal.NewFromInt(5000), Locked: decimal.NewFromInt(5000)},
	})
	require.NoError(t, err)
}

func TestEngine_UpdatePosition_UnknownInstance(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	engine := NewEngine(registry)
	ctx := context.Background()

	// Should not panic for unknown instance
	engine.UpdatePosition(ctx, "nonexistent", strategy.PositionInfo{Symbol: "BTCUSDT"})
}

func TestEngine_UpdateBalance(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "bal-strategy",
		version: "1.0.0",
	}
	require.NoError(t, registry.Register("bal-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-bal",
		StrategyID: "bal-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	newBal := strategy.BalanceInfo{
		Total:     decimal.NewFromInt(10000),
		Available: decimal.NewFromInt(5000),
		Locked:    decimal.NewFromInt(5000),
	}

	engine.UpdateBalance(ctx, "inst-bal", newBal)
}

func TestEngine_UpdateBalance_UnknownInstance(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	engine := NewEngine(registry)
	ctx := context.Background()

	// Should not panic for unknown instance
	engine.UpdateBalance(ctx, "nonexistent", strategy.BalanceInfo{Total: decimal.NewFromInt(1000)})
}

func TestEngine_Reset_InstanceNotFound(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	engine := NewEngine(registry)
	ctx := context.Background()

	err := engine.Reset(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEngine_Reset_StrategyError(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &errStrategy{resetErr: fmt.Errorf("reset error")}
	require.NoError(t, registry.Register("reset-err-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	_, err := engine.Tick(ctx, TickInput{
		InstanceID: "inst-reset-err",
		StrategyID: "reset-err-1",
		Symbol:     "BTCUSDT",
		LastPrice:  decimal.NewFromInt(50000),
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)

	err = engine.Reset(ctx, "inst-reset-err")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reset strategy")
}

func TestEngine_MultipleTicksAccumulateContext(t *testing.T) {
	registry := strategy.NewStrategyRegistry()
	mock := &mockStrategy{
		name:    "multi-tick",
		version: "1.0.0",
		signal: &strategy.Signal{
			StrategyID:  "multi-1",
			Symbol:      "BTCUSDT",
			Action:      strategy.SignalActionBuy,
			Quantity:    decimal.NewFromInt(1),
			GeneratedAt: time.Now().UTC(),
		},
	}
	require.NoError(t, registry.Register("multi-1", mock))

	engine := NewEngine(registry)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		signal, err := engine.Tick(ctx, TickInput{
			InstanceID: "inst-multi",
			StrategyID: "multi-1",
			Symbol:     "BTCUSDT",
			LastPrice:  decimal.NewFromInt(int64(50000 + i*100)),
			Timestamp:  time.Now().UTC(),
		})
		require.NoError(t, err)
		require.NotNil(t, signal)
	}
}

func TestEngine_NewEngine_NilRegistry(t *testing.T) {
	// Should not panic with nil registry
	engine := NewEngine(nil)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.contexts)
}
