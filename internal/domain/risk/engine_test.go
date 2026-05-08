package risk

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	publisher := events.NewPublisher()
	engine := NewEngine(publisher)

	require.NotNil(t, engine)
	assert.Equal(t, publisher, engine.publisher)
}

func TestEngine_ValidateOrder_SymbolNotTrading(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusMaintenance,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(10000),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.ErrorIs(t, err, ErrSymbolNotTrading)
}

func TestEngine_ValidateOrder_InvalidQuantity(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.0001), // Below MinQty
		Price:    decimal.NewFromInt(50000),
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(10000),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.ErrorIs(t, err, ErrInvalidOrderQuantity)
}

func TestEngine_ValidateOrder_InvalidPrice(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(200000), // Above MaxPrice
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(10000),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.ErrorIs(t, err, ErrInvalidOrderPrice)
}

func TestEngine_ValidateOrder_InvalidNotional(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.001),
		Price:    decimal.NewFromInt(5), // Notional = 0.005, below MinNotional
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(10000),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.ErrorIs(t, err, ErrInvalidNotional)
}

func TestEngine_ValidateOrder_DailyLossLimitExceeded(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		DailyPnL:        decimal.NewFromInt(-1500), // Exceeded limit
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(10000),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.ErrorIs(t, err, ErrDailyLossLimitExceeded)
}

func TestEngine_ValidateOrder_PositionLimitExceeded(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.6),
		Price:    decimal.NewFromInt(50000),
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(100000),
	}

	currentPosition := &entity.Position{
		Side:     entity.PositionSideLong,
		Quantity: decimal.NewFromFloat(0.5),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, currentPosition)
	assert.ErrorIs(t, err, ErrPositionLimitExceeded)
}

func TestEngine_ValidateOrder_InsufficientBalance(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(1000), // Insufficient for 0.1 * 50000 = 5000
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestEngine_ValidateOrder_Success(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy,
		Type:     entity.OrderTypeLimit,
		Quantity: decimal.NewFromFloat(0.1),
		Price:    decimal.NewFromInt(50000),
	}

	strategy := &entity.Strategy{
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		DailyPnL:        decimal.NewFromInt(-100),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
		MinPrice:    decimal.NewFromInt(1),
		MaxPrice:    decimal.NewFromInt(100000),
		TickSize:    decimal.NewFromInt(1),
		MinNotional: decimal.NewFromInt(10),
	}

	balance := &entity.Balance{
		Asset:     "USDT",
		Available: decimal.NewFromInt(10000),
	}

	err := engine.ValidateOrder(ctx, order, strategy, symbol, balance, nil)
	assert.NoError(t, err)
}

func TestEngine_ValidatePositionClose_FlatPosition(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideSell,
		Quantity: decimal.NewFromFloat(0.1),
	}

	position := &entity.Position{
		Side:     entity.PositionSideFlat,
		Quantity: decimal.Zero,
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
	}

	err := engine.ValidatePositionClose(ctx, order, position, symbol)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot close flat position")
}

func TestEngine_ValidatePositionClose_WrongSide(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideBuy, // Wrong side for closing long
		Quantity: decimal.NewFromFloat(0.1),
	}

	position := &entity.Position{
		Side:     entity.PositionSideLong,
		Quantity: decimal.NewFromFloat(0.5),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
	}

	err := engine.ValidatePositionClose(ctx, order, position, symbol)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order side must be opposite")
}

func TestEngine_ValidatePositionClose_ExceedsQuantity(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideSell,
		Quantity: decimal.NewFromFloat(1.0), // Exceeds position
	}

	position := &entity.Position{
		Side:     entity.PositionSideLong,
		Quantity: decimal.NewFromFloat(0.5),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
	}

	err := engine.ValidatePositionClose(ctx, order, position, symbol)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order quantity exceeds position quantity")
}

func TestEngine_ValidatePositionClose_Success(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	order := &entity.Order{
		Symbol:   "BTCUSDT",
		Side:     entity.OrderSideSell,
		Quantity: decimal.NewFromFloat(0.3),
	}

	position := &entity.Position{
		Side:     entity.PositionSideLong,
		Quantity: decimal.NewFromFloat(0.5),
	}

	symbol := &entity.Symbol{
		Name:        "BTCUSDT",
		Status:      entity.SymbolStatusTrading,
		MinOrderQty: decimal.NewFromFloat(0.001),
		MaxOrderQty: decimal.NewFromInt(100),
		StepSize:    decimal.NewFromFloat(0.001),
	}

	err := engine.ValidatePositionClose(ctx, order, position, symbol)
	assert.NoError(t, err)
}

func TestEngine_ShouldForceClose(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		status   entity.SymbolStatus
		expected bool
	}{
		{
			name:     "delisted symbol",
			status:   entity.SymbolStatusDelisted,
			expected: true,
		},
		{
			name:     "trading symbol",
			status:   entity.SymbolStatusTrading,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &entity.Symbol{
				Status: tt.status,
			}
			result := engine.ShouldForceClose(symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_ShouldBlockNewEntries(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name         string
		symbolStatus entity.SymbolStatus
		dailyPnL     decimal.Decimal
		maxDailyLoss decimal.Decimal
		expected     bool
	}{
		{
			name:         "symbol in maintenance",
			symbolStatus: entity.SymbolStatusMaintenance,
			dailyPnL:     decimal.Zero,
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     true,
		},
		{
			name:         "daily loss limit reached",
			symbolStatus: entity.SymbolStatusTrading,
			dailyPnL:     decimal.NewFromInt(-1500),
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     true,
		},
		{
			name:         "normal trading",
			symbolStatus: entity.SymbolStatusTrading,
			dailyPnL:     decimal.NewFromInt(-100),
			maxDailyLoss: decimal.NewFromInt(1000),
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbol := &entity.Symbol{
				Status: tt.symbolStatus,
			}
			strategy := &entity.Strategy{
				DailyPnL:     tt.dailyPnL,
				MaxDailyLoss: tt.maxDailyLoss,
			}
			result := engine.ShouldBlockNewEntries(strategy, symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_ErrorConstants(t *testing.T) {
	assert.NotNil(t, ErrPositionLimitExceeded)
	assert.NotNil(t, ErrDailyLossLimitExceeded)
	assert.NotNil(t, ErrInsufficientBalance)
	assert.NotNil(t, ErrSymbolNotTrading)
	assert.NotNil(t, ErrInvalidOrderQuantity)
	assert.NotNil(t, ErrInvalidOrderPrice)
	assert.NotNil(t, ErrInvalidNotional)
}

func TestEngine_PublishRiskViolation_WithPublisher(t *testing.T) {
	publisher := events.NewPublisher()
	engine := NewEngine(publisher)

	ctx := context.Background()
	strategyID := "strategy-1"
	violationType := "max_position_size"
	currentValue := decimal.NewFromInt(1000)
	limitValue := decimal.NewFromInt(500)
	action := "order_rejected"

	engine.publishRiskViolation(ctx, strategyID, violationType, currentValue, limitValue, action)

	// Give async publish time to complete
	time.Sleep(10 * time.Millisecond)
}

func TestEngine_PublishRiskViolation_NilPublisher(t *testing.T) {
	engine := NewEngine(nil)

	ctx := context.Background()
	strategyID := "strategy-1"
	violationType := "max_position_size"
	currentValue := decimal.NewFromInt(1000)
	limitValue := decimal.NewFromInt(500)
	action := "order_rejected"

	// Should not panic with nil publisher
	engine.publishRiskViolation(ctx, strategyID, violationType, currentValue, limitValue, action)
}

func TestEngine_PublishRiskWarning_WithPublisher(t *testing.T) {
	publisher := events.NewPublisher()
	engine := NewEngine(publisher)

	ctx := context.Background()
	strategyID := "strategy-1"
	warningType := "approaching_daily_loss_limit"
	currentValue := decimal.NewFromInt(400)
	limitValue := decimal.NewFromInt(500)
	thresholdPct := decimal.NewFromInt(80)

	engine.publishRiskWarning(ctx, strategyID, warningType, currentValue, limitValue, thresholdPct)

	// Give async publish time to complete
	time.Sleep(10 * time.Millisecond)
}

func TestEngine_PublishRiskWarning_NilPublisher(t *testing.T) {
	engine := NewEngine(nil)

	ctx := context.Background()
	strategyID := "strategy-1"
	warningType := "approaching_daily_loss_limit"
	currentValue := decimal.NewFromInt(400)
	limitValue := decimal.NewFromInt(500)
	thresholdPct := decimal.NewFromInt(80)

	// Should not panic with nil publisher
	engine.publishRiskWarning(ctx, strategyID, warningType, currentValue, limitValue, thresholdPct)
}
