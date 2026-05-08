package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/tui/components"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBridge(t *testing.T) {
	// Create a minimal program for testing
	m := &Model{}
	p := tea.NewProgram(m)

	eb := NewEventBridge(p)

	require.NotNil(t, eb)
	assert.Equal(t, p, eb.program)
}

func TestEventBridge_OnEvent_NilProgram(t *testing.T) {
	eb := &EventBridge{program: nil}
	ctx := context.Background()

	event := &events.OrderCreatedEvent{
		OrderID:  "order1",
		Symbol:   "BTCUSDT",
		Side:     "BUY",
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	// Should not panic with nil program
	assert.NotPanics(t, func() {
		eb.OnEvent(ctx, event)
	})
}

func TestEventBridge_ConvertEvent_OrderCreated(t *testing.T) {
	eb := &EventBridge{}

	event := &events.OrderCreatedEvent{
		OrderID:  "order1",
		Symbol:   "BTCUSDT",
		Side:     "BUY",
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	orderMsg, ok := msg.(orderEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelInfo, orderMsg.level)
	assert.Contains(t, orderMsg.message, "Order created")
	assert.Contains(t, orderMsg.message, "BUY")
	assert.Contains(t, orderMsg.message, "BTCUSDT")
}

func TestEventBridge_ConvertEvent_OrderSubmitted(t *testing.T) {
	eb := &EventBridge{}

	event := &events.OrderSubmittedEvent{
		OrderID:         "order1",
		ExchangeOrderID: "exchange123",
		SubmittedAtUTC:  time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	orderMsg, ok := msg.(orderEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelSuccess, orderMsg.level)
	assert.Contains(t, orderMsg.message, "Order submitted")
	assert.Contains(t, orderMsg.message, "exchange123")
}

func TestEventBridge_ConvertEvent_OrderFilled(t *testing.T) {
	eb := &EventBridge{}

	event := &events.OrderFilledEvent{
		OrderID:      "order1",
		FilledQty:    decimal.NewFromFloat(0.1),
		AvgFillPrice: decimal.NewFromInt(50000),
		FilledAtUTC:  time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	orderMsg, ok := msg.(orderEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelSuccess, orderMsg.level)
	assert.Contains(t, orderMsg.message, "Order filled")
	assert.Contains(t, orderMsg.message, "0.1")
	assert.Contains(t, orderMsg.message, "50000")
}

func TestEventBridge_ConvertEvent_OrderPartial(t *testing.T) {
	eb := &EventBridge{}

	event := &events.OrderPartialEvent{
		OrderID:      "order1",
		FilledQty:    decimal.NewFromFloat(0.05),
		RemainingQty: decimal.NewFromFloat(0.05),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	orderMsg, ok := msg.(orderEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelInfo, orderMsg.level)
	assert.Contains(t, orderMsg.message, "partial fill")
	assert.Contains(t, orderMsg.message, "0.05")
}

func TestEventBridge_ConvertEvent_OrderCanceled(t *testing.T) {
	eb := &EventBridge{}

	event := &events.OrderCanceledEvent{
		OrderID:       "order1",
		Reason:        "User requested",
		CanceledAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	orderMsg, ok := msg.(orderEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelWarning, orderMsg.level)
	assert.Contains(t, orderMsg.message, "Order canceled")
	assert.Contains(t, orderMsg.message, "User requested")
}

func TestEventBridge_ConvertEvent_OrderRejected(t *testing.T) {
	eb := &EventBridge{}

	event := &events.OrderRejectedEvent{
		OrderID:       "order1",
		Reason:        "Insufficient balance",
		RejectedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	orderMsg, ok := msg.(orderEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelError, orderMsg.level)
	assert.Contains(t, orderMsg.message, "Order rejected")
	assert.Contains(t, orderMsg.message, "Insufficient balance")
}

func TestEventBridge_ConvertEvent_PositionOpened(t *testing.T) {
	eb := &EventBridge{}

	event := &events.PositionOpenedEvent{
		PositionID:  "pos1",
		Symbol:      "BTCUSDT",
		Side:        "LONG",
		Quantity:    decimal.NewFromFloat(0.1),
		EntryPrice:  decimal.NewFromInt(50000),
		OpenedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	posMsg, ok := msg.(positionEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelSuccess, posMsg.level)
	assert.Contains(t, posMsg.message, "Position opened")
	assert.Contains(t, posMsg.message, "LONG")
	assert.Contains(t, posMsg.message, "BTCUSDT")
}

func TestEventBridge_ConvertEvent_PositionUpdated(t *testing.T) {
	eb := &EventBridge{}

	event := &events.PositionUpdatedEvent{
		PositionID:    "pos1",
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(100),
		UpdatedAtUTC:  time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	posMsg, ok := msg.(positionEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelInfo, posMsg.level)
	assert.Contains(t, posMsg.message, "Position updated")
	assert.Contains(t, posMsg.message, "100")
}

func TestEventBridge_ConvertEvent_PositionClosed(t *testing.T) {
	eb := &EventBridge{}

	event := &events.PositionClosedEvent{
		PositionID:  "pos1",
		Symbol:      "BTCUSDT",
		RealizedPnL: decimal.NewFromInt(100),
		ClosedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	posMsg, ok := msg.(positionEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelSuccess, posMsg.level)
	assert.Contains(t, posMsg.message, "Position closed")
	assert.Contains(t, posMsg.message, "BTCUSDT")
	assert.Contains(t, posMsg.message, "100")
}

func TestEventBridge_ConvertEvent_BalanceUpdated(t *testing.T) {
	eb := &EventBridge{}

	event := &events.BalanceUpdatedEvent{
		Asset:        "USDT",
		Total:        decimal.NewFromInt(10000),
		Available:    decimal.NewFromInt(9500),
		Locked:       decimal.NewFromInt(500),
		UpdatedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	balMsg, ok := msg.(balanceEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelInfo, balMsg.level)
	assert.Contains(t, balMsg.message, "Balance updated")
	assert.Contains(t, balMsg.message, "USDT")
	assert.Contains(t, balMsg.message, "10000")
}

func TestEventBridge_ConvertEvent_RiskViolation(t *testing.T) {
	eb := &EventBridge{}

	event := &events.RiskViolationEvent{
		ViolationType: "max_position_size",
		Action:        "order_rejected",
		CurrentValue:  decimal.NewFromInt(1500),
		LimitValue:    decimal.NewFromInt(1000),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	riskMsg, ok := msg.(riskEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelError, riskMsg.level)
	assert.Contains(t, riskMsg.message, "Risk violation")
	assert.Contains(t, riskMsg.message, "max_position_size")
	assert.Contains(t, riskMsg.message, "order_rejected")
}

func TestEventBridge_ConvertEvent_RiskWarning(t *testing.T) {
	eb := &EventBridge{}

	event := &events.RiskWarningEvent{
		WarningType:  "daily_loss_threshold",
		ThresholdPct: decimal.NewFromInt(80),
		CurrentValue: decimal.NewFromInt(400),
		LimitValue:   decimal.NewFromInt(500),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	riskMsg, ok := msg.(riskEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelWarning, riskMsg.level)
	assert.Contains(t, riskMsg.message, "Risk warning")
	assert.Contains(t, riskMsg.message, "daily_loss_threshold")
	assert.Contains(t, riskMsg.message, "80")
}

func TestEventBridge_ConvertEvent_StrategyStarted(t *testing.T) {
	eb := &EventBridge{}

	event := &events.StrategyStartedEvent{
		StrategyID:   "strat1",
		StrategyName: "SMA-Cross",
		StrategyType: "trend_following",
		StartedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	stratMsg, ok := msg.(strategyEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelSuccess, stratMsg.level)
	assert.Contains(t, stratMsg.message, "Strategy started")
	assert.Contains(t, stratMsg.message, "SMA-Cross")
	assert.Contains(t, stratMsg.message, "trend_following")
}

func TestEventBridge_ConvertEvent_StrategyPaused(t *testing.T) {
	eb := &EventBridge{}

	event := &events.StrategyPausedEvent{
		StrategyID:  "strat1",
		Reason:      "Risk limit reached",
		PausedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	stratMsg, ok := msg.(strategyEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelWarning, stratMsg.level)
	assert.Contains(t, stratMsg.message, "Strategy paused")
	assert.Contains(t, stratMsg.message, "Risk limit reached")
}

func TestEventBridge_ConvertEvent_StrategyStopped(t *testing.T) {
	eb := &EventBridge{}

	event := &events.StrategyStoppedEvent{
		StrategyID:   "strat1",
		Reason:       "User requested",
		FinalPnL:     decimal.NewFromInt(500),
		StoppedAtUTC: time.Now().UTC(),
	}

	msg := eb.convertEvent(event)

	require.NotNil(t, msg)
	stratMsg, ok := msg.(strategyEventMsg)
	require.True(t, ok)
	assert.Equal(t, components.LogLevelInfo, stratMsg.level)
	assert.Contains(t, stratMsg.message, "Strategy stopped")
	assert.Contains(t, stratMsg.message, "User requested")
	assert.Contains(t, stratMsg.message, "500")
}

func TestEventBridge_ConvertEvent_UnknownEvent(t *testing.T) {
	eb := &EventBridge{}

	// Create a mock unknown event type
	type unknownEvent struct {
		events.BaseEvent
	}

	event := &unknownEvent{}

	msg := eb.convertEvent(event)

	assert.Nil(t, msg)
}

func TestEventBridge_ConvertEvent_AllEventTypes(t *testing.T) {
	eb := &EventBridge{}

	events := []events.Event{
		&events.OrderCreatedEvent{OrderID: "1", Symbol: "BTCUSDT", Side: "BUY", Quantity: decimal.NewFromInt(1), Price: decimal.NewFromInt(50000)},
		&events.OrderSubmittedEvent{OrderID: "1", ExchangeOrderID: "ex1"},
		&events.OrderFilledEvent{OrderID: "1", FilledQty: decimal.NewFromInt(1), AvgFillPrice: decimal.NewFromInt(50000)},
		&events.OrderPartialEvent{OrderID: "1", FilledQty: decimal.NewFromFloat(0.5), RemainingQty: decimal.NewFromFloat(0.5)},
		&events.OrderCanceledEvent{OrderID: "1", Reason: "test"},
		&events.OrderRejectedEvent{OrderID: "1", Reason: "test"},
		&events.PositionOpenedEvent{PositionID: "1", Symbol: "BTCUSDT", Side: "LONG", Quantity: decimal.NewFromInt(1), EntryPrice: decimal.NewFromInt(50000)},
		&events.PositionUpdatedEvent{PositionID: "1", CurrentPrice: decimal.NewFromInt(51000), UnrealizedPnL: decimal.NewFromInt(100)},
		&events.PositionClosedEvent{PositionID: "1", Symbol: "BTCUSDT", RealizedPnL: decimal.NewFromInt(100)},
		&events.BalanceUpdatedEvent{Asset: "USDT", Total: decimal.NewFromInt(10000), Available: decimal.NewFromInt(9500), Locked: decimal.NewFromInt(500)},
		&events.RiskViolationEvent{ViolationType: "test", Action: "test", CurrentValue: decimal.NewFromInt(100), LimitValue: decimal.NewFromInt(50)},
		&events.RiskWarningEvent{WarningType: "test", ThresholdPct: decimal.NewFromInt(80), CurrentValue: decimal.NewFromInt(80), LimitValue: decimal.NewFromInt(100)},
		&events.StrategyStartedEvent{StrategyID: "1", StrategyName: "test", StrategyType: "test"},
		&events.StrategyPausedEvent{StrategyID: "1", Reason: "test"},
		&events.StrategyStoppedEvent{StrategyID: "1", Reason: "test", FinalPnL: decimal.NewFromInt(100)},
	}

	for _, event := range events {
		msg := eb.convertEvent(event)
		assert.NotNil(t, msg, "Event type %T should produce a message", event)
	}
}
