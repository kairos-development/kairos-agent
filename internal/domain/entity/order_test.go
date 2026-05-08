package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestOrder_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   OrderStatus
		expected bool
	}{
		{
			name:     "filled is terminal",
			status:   OrderStatusFilled,
			expected: true,
		},
		{
			name:     "canceled is terminal",
			status:   OrderStatusCanceled,
			expected: true,
		},
		{
			name:     "rejected is terminal",
			status:   OrderStatusRejected,
			expected: true,
		},
		{
			name:     "pending is not terminal",
			status:   OrderStatusPending,
			expected: false,
		},
		{
			name:     "submitted is not terminal",
			status:   OrderStatusSubmitted,
			expected: false,
		},
		{
			name:     "partial is not terminal",
			status:   OrderStatusPartial,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{Status: tt.status}
			assert.Equal(t, tt.expected, order.IsTerminal())
		})
	}
}

func TestOrder_IsPartiallyFilled(t *testing.T) {
	tests := []struct {
		name      string
		status    OrderStatus
		filledQty decimal.Decimal
		expected  bool
	}{
		{
			name:      "partial with filled qty",
			status:    OrderStatusPartial,
			filledQty: decimal.NewFromFloat(0.5),
			expected:  true,
		},
		{
			name:      "partial with zero filled qty",
			status:    OrderStatusPartial,
			filledQty: decimal.Zero,
			expected:  false,
		},
		{
			name:      "submitted with filled qty",
			status:    OrderStatusSubmitted,
			filledQty: decimal.NewFromFloat(0.5),
			expected:  false,
		},
		{
			name:      "filled status",
			status:    OrderStatusFilled,
			filledQty: decimal.NewFromInt(1),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				Status:    tt.status,
				FilledQty: tt.filledQty,
			}
			assert.Equal(t, tt.expected, order.IsPartiallyFilled())
		})
	}
}

func TestOrder_IsPending(t *testing.T) {
	tests := []struct {
		name     string
		status   OrderStatus
		expected bool
	}{
		{
			name:     "pending status",
			status:   OrderStatusPending,
			expected: true,
		},
		{
			name:     "submitted status",
			status:   OrderStatusSubmitted,
			expected: false,
		},
		{
			name:     "filled status",
			status:   OrderStatusFilled,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{Status: tt.status}
			assert.Equal(t, tt.expected, order.IsPending())
		})
	}
}

func TestOrder_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   OrderStatus
		expected bool
	}{
		{
			name:     "submitted is active",
			status:   OrderStatusSubmitted,
			expected: true,
		},
		{
			name:     "partial is active",
			status:   OrderStatusPartial,
			expected: true,
		},
		{
			name:     "pending is not active",
			status:   OrderStatusPending,
			expected: false,
		},
		{
			name:     "filled is not active",
			status:   OrderStatusFilled,
			expected: false,
		},
		{
			name:     "canceled is not active",
			status:   OrderStatusCanceled,
			expected: false,
		},
		{
			name:     "rejected is not active",
			status:   OrderStatusRejected,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{Status: tt.status}
			assert.Equal(t, tt.expected, order.IsActive())
		})
	}
}

func TestOrder_Fields(t *testing.T) {
	now := time.Now().UTC()
	submittedAt := now.Add(-1 * time.Hour)
	filledAt := now.Add(-30 * time.Minute)

	order := &Order{
		ID:              "order123",
		ClientOrderID:   "client456",
		ExchangeOrderID: "exchange789",
		StrategyID:      "strat1",
		Symbol:          "BTCUSDT",
		Side:            OrderSideBuy,
		Type:            OrderTypeLimit,
		Status:          OrderStatusFilled,
		TimeInForce:     TimeInForceGTC,
		Quantity:        decimal.NewFromFloat(0.1),
		Price:           decimal.NewFromInt(50000),
		FilledQty:       decimal.NewFromFloat(0.1),
		RemainingQty:    decimal.Zero,
		AvgFillPrice:    decimal.NewFromInt(50000),
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
		SubmittedAtUTC:  &submittedAt,
		FilledAtUTC:     &filledAt,
	}

	assert.Equal(t, "order123", order.ID)
	assert.Equal(t, "client456", order.ClientOrderID)
	assert.Equal(t, "exchange789", order.ExchangeOrderID)
	assert.Equal(t, "strat1", order.StrategyID)
	assert.Equal(t, "BTCUSDT", order.Symbol)
	assert.Equal(t, OrderSideBuy, order.Side)
	assert.Equal(t, OrderTypeLimit, order.Type)
	assert.Equal(t, OrderStatusFilled, order.Status)
	assert.Equal(t, TimeInForceGTC, order.TimeInForce)
	assert.True(t, order.Quantity.Equal(decimal.NewFromFloat(0.1)))
	assert.True(t, order.Price.Equal(decimal.NewFromInt(50000)))
	assert.True(t, order.FilledQty.Equal(decimal.NewFromFloat(0.1)))
	assert.True(t, order.RemainingQty.Equal(decimal.Zero))
	assert.True(t, order.AvgFillPrice.Equal(decimal.NewFromInt(50000)))
	assert.Equal(t, now, order.CreatedAtUTC)
	assert.Equal(t, now, order.UpdatedAtUTC)
	assert.NotNil(t, order.SubmittedAtUTC)
	assert.NotNil(t, order.FilledAtUTC)
}

func TestOrderSide_Constants(t *testing.T) {
	assert.Equal(t, OrderSide("buy"), OrderSideBuy)
	assert.Equal(t, OrderSide("sell"), OrderSideSell)
}

func TestOrderType_Constants(t *testing.T) {
	assert.Equal(t, OrderType("limit"), OrderTypeLimit)
	assert.Equal(t, OrderType("market"), OrderTypeMarket)
}

func TestOrderStatus_Constants(t *testing.T) {
	assert.Equal(t, OrderStatus("pending"), OrderStatusPending)
	assert.Equal(t, OrderStatus("submitted"), OrderStatusSubmitted)
	assert.Equal(t, OrderStatus("partial"), OrderStatusPartial)
	assert.Equal(t, OrderStatus("filled"), OrderStatusFilled)
	assert.Equal(t, OrderStatus("canceled"), OrderStatusCanceled)
	assert.Equal(t, OrderStatus("rejected"), OrderStatusRejected)
}

func TestTimeInForce_Constants(t *testing.T) {
	assert.Equal(t, TimeInForce("GTC"), TimeInForceGTC)
	assert.Equal(t, TimeInForce("IOC"), TimeInForceIOC)
	assert.Equal(t, TimeInForce("FOK"), TimeInForceFOK)
}
