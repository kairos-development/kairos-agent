package sqlite

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tmpFile, err := os.CreateTemp("", "test_orders_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	db, err := Open(tmpFile.Name())
	require.NoError(t, err)

	// Create orders table
	_, err = db.Conn().Exec(`
		CREATE TABLE orders (
			id TEXT PRIMARY KEY,
			client_order_id TEXT NOT NULL UNIQUE,
			exchange_order_id TEXT,
			strategy_id TEXT NOT NULL,
			symbol TEXT NOT NULL,
			side TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			time_in_force TEXT NOT NULL,
			quantity TEXT NOT NULL,
			price TEXT NOT NULL,
			filled_qty TEXT NOT NULL,
			remaining_qty TEXT NOT NULL,
			avg_fill_price TEXT NOT NULL,
			created_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			submitted_at_utc TEXT,
			filled_at_utc TEXT
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestNewOrderRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestOrderRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:            "order1",
		ClientOrderID: "client1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusPending,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, order)
	assert.NoError(t, err)

	// Verify order was created
	retrieved, err := repo.GetByID(ctx, "order1")
	require.NoError(t, err)
	assert.Equal(t, "order1", retrieved.ID)
	assert.Equal(t, "client1", retrieved.ClientOrderID)
	assert.Equal(t, "BTCUSDT", retrieved.Symbol)
	assert.True(t, retrieved.Quantity.Equal(decimal.NewFromFloat(0.1)))
}

func TestOrderRepository_Create_DuplicateKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:            "order1",
		ClientOrderID: "client1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusPending,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	// Try to create duplicate
	err = repo.Create(ctx, order)
	assert.ErrorIs(t, err, ErrDuplicateKey)
}

func TestOrderRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:            "order1",
		ClientOrderID: "client1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusPending,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	// Update order
	order.Status = entity.OrderStatusSubmitted
	order.ExchangeOrderID = "exchange1"
	order.UpdatedAtUTC = now.Add(1 * time.Second)
	submittedAt := now.Add(1 * time.Second)
	order.SubmittedAtUTC = &submittedAt

	err = repo.Update(ctx, order)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, "order1")
	require.NoError(t, err)
	assert.Equal(t, entity.OrderStatusSubmitted, retrieved.Status)
	assert.Equal(t, "exchange1", retrieved.ExchangeOrderID)
	assert.NotNil(t, retrieved.SubmittedAtUTC)
}

func TestOrderRepository_Update_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:           "nonexistent",
		Status:       entity.OrderStatusSubmitted,
		UpdatedAtUTC: now,
	}

	err := repo.Update(ctx, order)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOrderRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:            "order1",
		ClientOrderID: "client1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusPending,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, "order1")
	require.NoError(t, err)
	assert.Equal(t, "order1", retrieved.ID)
	assert.Equal(t, "client1", retrieved.ClientOrderID)
}

func TestOrderRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOrderRepository_GetByClientOrderID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:            "order1",
		ClientOrderID: "client1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusPending,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	retrieved, err := repo.GetByClientOrderID(ctx, "client1")
	require.NoError(t, err)
	assert.Equal(t, "order1", retrieved.ID)
	assert.Equal(t, "client1", retrieved.ClientOrderID)
}

func TestOrderRepository_GetByExchangeOrderID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	order := &entity.Order{
		ID:              "order1",
		ClientOrderID:   "client1",
		ExchangeOrderID: "exchange1",
		StrategyID:      "strategy1",
		Symbol:          "BTCUSDT",
		Side:            entity.OrderSideBuy,
		Type:            entity.OrderTypeLimit,
		Status:          entity.OrderStatusSubmitted,
		TimeInForce:     entity.TimeInForceGTC,
		Quantity:        decimal.NewFromFloat(0.1),
		Price:           decimal.NewFromInt(50000),
		FilledQty:       decimal.Zero,
		RemainingQty:    decimal.NewFromFloat(0.1),
		AvgFillPrice:    decimal.Zero,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	retrieved, err := repo.GetByExchangeOrderID(ctx, "exchange1")
	require.NoError(t, err)
	assert.Equal(t, "order1", retrieved.ID)
	assert.Equal(t, "exchange1", retrieved.ExchangeOrderID)
}

func TestOrderRepository_ListByStrategy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create multiple orders for same strategy
	for i := 1; i <= 3; i++ {
		order := &entity.Order{
			ID:            "order" + string(rune('0'+i)),
			ClientOrderID: "client" + string(rune('0'+i)),
			StrategyID:    "strategy1",
			Symbol:        "BTCUSDT",
			Side:          entity.OrderSideBuy,
			Type:          entity.OrderTypeLimit,
			Status:        entity.OrderStatusPending,
			TimeInForce:   entity.TimeInForceGTC,
			Quantity:      decimal.NewFromFloat(0.1),
			Price:         decimal.NewFromInt(50000),
			FilledQty:     decimal.Zero,
			RemainingQty:  decimal.NewFromFloat(0.1),
			AvgFillPrice:  decimal.Zero,
			CreatedAtUTC:  now.Add(time.Duration(i) * time.Second),
			UpdatedAtUTC:  now.Add(time.Duration(i) * time.Second),
		}
		err := repo.Create(ctx, order)
		require.NoError(t, err)
	}

	orders, err := repo.ListByStrategy(ctx, "strategy1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, orders, 3)
}

func TestOrderRepository_ListActive(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create orders with different statuses
	statuses := []entity.OrderStatus{
		entity.OrderStatusPending,
		entity.OrderStatusSubmitted,
		entity.OrderStatusPartial,
		entity.OrderStatusFilled,
	}

	for i, status := range statuses {
		order := &entity.Order{
			ID:            "order" + string(rune('0'+i+1)),
			ClientOrderID: "client" + string(rune('0'+i+1)),
			StrategyID:    "strategy1",
			Symbol:        "BTCUSDT",
			Side:          entity.OrderSideBuy,
			Type:          entity.OrderTypeLimit,
			Status:        status,
			TimeInForce:   entity.TimeInForceGTC,
			Quantity:      decimal.NewFromFloat(0.1),
			Price:         decimal.NewFromInt(50000),
			FilledQty:     decimal.Zero,
			RemainingQty:  decimal.NewFromFloat(0.1),
			AvgFillPrice:  decimal.Zero,
			CreatedAtUTC:  now,
			UpdatedAtUTC:  now,
		}
		err := repo.Create(ctx, order)
		require.NoError(t, err)
	}

	// ListActive should return only submitted and partial
	orders, err := repo.ListActive(ctx)
	require.NoError(t, err)
	assert.Len(t, orders, 2)
	for _, order := range orders {
		assert.True(t, order.Status == entity.OrderStatusSubmitted || order.Status == entity.OrderStatusPartial)
	}
}

func TestOrderRepository_ListInFlight(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewOrderRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	since := now.Add(-1 * time.Hour)

	// Create orders before and after 'since'
	oldOrder := &entity.Order{
		ID:            "order1",
		ClientOrderID: "client1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusPending,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  since.Add(-1 * time.Minute),
		UpdatedAtUTC:  since.Add(-1 * time.Minute),
	}

	newOrder := &entity.Order{
		ID:            "order2",
		ClientOrderID: "client2",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.OrderSideBuy,
		Type:          entity.OrderTypeLimit,
		Status:        entity.OrderStatusSubmitted,
		TimeInForce:   entity.TimeInForceGTC,
		Quantity:      decimal.NewFromFloat(0.1),
		Price:         decimal.NewFromInt(50000),
		FilledQty:     decimal.Zero,
		RemainingQty:  decimal.NewFromFloat(0.1),
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  since.Add(1 * time.Minute),
		UpdatedAtUTC:  since.Add(1 * time.Minute),
	}

	err := repo.Create(ctx, oldOrder)
	require.NoError(t, err)
	err = repo.Create(ctx, newOrder)
	require.NoError(t, err)

	// ListInFlight should return only orders created after 'since'
	orders, err := repo.ListInFlight(ctx, since)
	require.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, "order2", orders[0].ID)
}

func TestNullString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected sql.NullString
	}{
		{
			name:     "empty string",
			input:    "",
			expected: sql.NullString{String: "", Valid: false},
		},
		{
			name:     "non-empty string",
			input:    "test",
			expected: sql.NullString{String: "test", Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNullTime(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name  string
		input *time.Time
		valid bool
	}{
		{
			name:  "nil time",
			input: nil,
			valid: false,
		},
		{
			name:  "non-nil time",
			input: &now,
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullTime(tt.input)
			assert.Equal(t, tt.valid, result.Valid)
			if tt.valid {
				assert.NotEmpty(t, result.String)
			}
		})
	}
}
