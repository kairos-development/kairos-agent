package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSymbolTestDB(t *testing.T) (*DB, func()) {
	tmpFile, err := os.CreateTemp("", "test_symbols_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	db, err := Open(tmpFile.Name())
	require.NoError(t, err)

	// Create symbols table
	_, err = db.Conn().Exec(`
		CREATE TABLE symbols (
			name TEXT PRIMARY KEY,
			base_currency TEXT NOT NULL,
			quote_currency TEXT NOT NULL,
			status TEXT NOT NULL,
			min_order_qty TEXT NOT NULL,
			max_order_qty TEXT NOT NULL,
			min_price TEXT NOT NULL,
			max_price TEXT NOT NULL,
			tick_size TEXT NOT NULL,
			step_size TEXT NOT NULL,
			min_notional TEXT NOT NULL,
			maker_fee TEXT NOT NULL,
			taker_fee TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestNewSymbolRepository(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestSymbolRepository_Upsert_Insert(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	symbol := &entity.Symbol{
		Name:          "BTCUSDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   decimal.NewFromFloat(0.001),
		MaxOrderQty:   decimal.NewFromInt(100),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromInt(1),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
		MakerFee:      decimal.NewFromFloat(0.001),
		TakerFee:      decimal.NewFromFloat(0.002),
		UpdatedAtUTC:  now,
	}

	err := repo.Upsert(ctx, symbol)
	assert.NoError(t, err)

	// Verify symbol was inserted
	retrieved, err := repo.GetByName(ctx, "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, "BTCUSDT", retrieved.Name)
	assert.Equal(t, "BTC", retrieved.BaseCurrency)
	assert.Equal(t, entity.SymbolStatusTrading, retrieved.Status)
}

func TestSymbolRepository_Upsert_Update(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert initial symbol
	symbol := &entity.Symbol{
		Name:          "BTCUSDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   decimal.NewFromFloat(0.001),
		MaxOrderQty:   decimal.NewFromInt(100),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromInt(1),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
		MakerFee:      decimal.NewFromFloat(0.001),
		TakerFee:      decimal.NewFromFloat(0.002),
		UpdatedAtUTC:  now,
	}

	err := repo.Upsert(ctx, symbol)
	require.NoError(t, err)

	// Update symbol
	symbol.Status = entity.SymbolStatusMaintenance
	symbol.MaxOrderQty = decimal.NewFromInt(50)
	symbol.UpdatedAtUTC = now.Add(1 * time.Hour)

	err = repo.Upsert(ctx, symbol)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByName(ctx, "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, entity.SymbolStatusMaintenance, retrieved.Status)
	assert.True(t, retrieved.MaxOrderQty.Equal(decimal.NewFromInt(50)))
}

func TestSymbolRepository_GetByName(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	symbol := &entity.Symbol{
		Name:          "BTCUSDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   decimal.NewFromFloat(0.001),
		MaxOrderQty:   decimal.NewFromInt(100),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromInt(1),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
		MakerFee:      decimal.NewFromFloat(0.001),
		TakerFee:      decimal.NewFromFloat(0.002),
		UpdatedAtUTC:  now,
	}

	err := repo.Upsert(ctx, symbol)
	require.NoError(t, err)

	retrieved, err := repo.GetByName(ctx, "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, "BTCUSDT", retrieved.Name)
	assert.Equal(t, "BTC", retrieved.BaseCurrency)
	assert.Equal(t, "USDT", retrieved.QuoteCurrency)
}

func TestSymbolRepository_GetByName_NotFound(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "NONEXISTENT")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSymbolRepository_List(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert multiple symbols
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	for _, name := range symbols {
		symbol := &entity.Symbol{
			Name:          name,
			BaseCurrency:  name[:3],
			QuoteCurrency: "USDT",
			Status:        entity.SymbolStatusTrading,
			MinOrderQty:   decimal.NewFromFloat(0.001),
			MaxOrderQty:   decimal.NewFromInt(100),
			MinPrice:      decimal.NewFromInt(1),
			MaxPrice:      decimal.NewFromInt(100000),
			TickSize:      decimal.NewFromInt(1),
			StepSize:      decimal.NewFromFloat(0.001),
			MinNotional:   decimal.NewFromInt(10),
			MakerFee:      decimal.NewFromFloat(0.001),
			TakerFee:      decimal.NewFromFloat(0.002),
			UpdatedAtUTC:  now,
		}
		err := repo.Upsert(ctx, symbol)
		require.NoError(t, err)
	}

	// List all symbols
	retrieved, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)

	// Verify ordering by name
	assert.Equal(t, "BTCUSDT", retrieved[0].Name)
	assert.Equal(t, "ETHUSDT", retrieved[1].Name)
	assert.Equal(t, "SOLUSDT", retrieved[2].Name)
}

func TestSymbolRepository_List_Empty(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()

	symbols, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, symbols)
}

func TestSymbolRepository_ListByStatus(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert symbols with different statuses
	statuses := []entity.SymbolStatus{
		entity.SymbolStatusTrading,
		entity.SymbolStatusTrading,
		entity.SymbolStatusMaintenance,
		entity.SymbolStatusSuspended,
	}

	for i, status := range statuses {
		symbol := &entity.Symbol{
			Name:          "SYMBOL" + string(rune('0'+i+1)),
			BaseCurrency:  "BASE",
			QuoteCurrency: "QUOTE",
			Status:        status,
			MinOrderQty:   decimal.NewFromFloat(0.001),
			MaxOrderQty:   decimal.NewFromInt(100),
			MinPrice:      decimal.NewFromInt(1),
			MaxPrice:      decimal.NewFromInt(100000),
			TickSize:      decimal.NewFromInt(1),
			StepSize:      decimal.NewFromFloat(0.001),
			MinNotional:   decimal.NewFromInt(10),
			MakerFee:      decimal.NewFromFloat(0.001),
			TakerFee:      decimal.NewFromFloat(0.002),
			UpdatedAtUTC:  now,
		}
		err := repo.Upsert(ctx, symbol)
		require.NoError(t, err)
	}

	// List only trading symbols
	trading, err := repo.ListByStatus(ctx, entity.SymbolStatusTrading)
	require.NoError(t, err)
	assert.Len(t, trading, 2)
	for _, symbol := range trading {
		assert.Equal(t, entity.SymbolStatusTrading, symbol.Status)
	}

	// List only maintenance symbols
	maintenance, err := repo.ListByStatus(ctx, entity.SymbolStatusMaintenance)
	require.NoError(t, err)
	assert.Len(t, maintenance, 1)
	assert.Equal(t, entity.SymbolStatusMaintenance, maintenance[0].Status)
}

func TestSymbolRepository_ListByStatus_Empty(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()

	symbols, err := repo.ListByStatus(ctx, entity.SymbolStatusTrading)
	require.NoError(t, err)
	assert.Empty(t, symbols)
}

func TestSymbolRepository_DecimalPrecision(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Use high precision decimal values
	minQty, _ := decimal.NewFromString("0.00000001")
	makerFee, _ := decimal.NewFromString("0.00075")
	takerFee, _ := decimal.NewFromString("0.00085")

	symbol := &entity.Symbol{
		Name:          "BTCUSDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   minQty,
		MaxOrderQty:   decimal.NewFromInt(100),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromInt(1),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
		MakerFee:      makerFee,
		TakerFee:      takerFee,
		UpdatedAtUTC:  now,
	}

	err := repo.Upsert(ctx, symbol)
	require.NoError(t, err)

	// Verify precision is preserved
	retrieved, err := repo.GetByName(ctx, "BTCUSDT")
	require.NoError(t, err)
	assert.True(t, retrieved.MinOrderQty.Equal(minQty))
	assert.True(t, retrieved.MakerFee.Equal(makerFee))
	assert.True(t, retrieved.TakerFee.Equal(takerFee))
}

func TestSymbolRepository_AllFields(t *testing.T) {
	db, cleanup := setupSymbolTestDB(t)
	defer cleanup()

	repo := NewSymbolRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	symbol := &entity.Symbol{
		Name:          "BTCUSDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Status:        entity.SymbolStatusTrading,
		MinOrderQty:   decimal.NewFromFloat(0.001),
		MaxOrderQty:   decimal.NewFromInt(100),
		MinPrice:      decimal.NewFromInt(1),
		MaxPrice:      decimal.NewFromInt(100000),
		TickSize:      decimal.NewFromInt(1),
		StepSize:      decimal.NewFromFloat(0.001),
		MinNotional:   decimal.NewFromInt(10),
		MakerFee:      decimal.NewFromFloat(0.001),
		TakerFee:      decimal.NewFromFloat(0.002),
		UpdatedAtUTC:  now,
	}

	err := repo.Upsert(ctx, symbol)
	require.NoError(t, err)

	// Verify all fields
	retrieved, err := repo.GetByName(ctx, "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, symbol.Name, retrieved.Name)
	assert.Equal(t, symbol.BaseCurrency, retrieved.BaseCurrency)
	assert.Equal(t, symbol.QuoteCurrency, retrieved.QuoteCurrency)
	assert.Equal(t, symbol.Status, retrieved.Status)
	assert.True(t, symbol.MinOrderQty.Equal(retrieved.MinOrderQty))
	assert.True(t, symbol.MaxOrderQty.Equal(retrieved.MaxOrderQty))
	assert.True(t, symbol.MinPrice.Equal(retrieved.MinPrice))
	assert.True(t, symbol.MaxPrice.Equal(retrieved.MaxPrice))
	assert.True(t, symbol.TickSize.Equal(retrieved.TickSize))
	assert.True(t, symbol.StepSize.Equal(retrieved.StepSize))
	assert.True(t, symbol.MinNotional.Equal(retrieved.MinNotional))
	assert.True(t, symbol.MakerFee.Equal(retrieved.MakerFee))
	assert.True(t, symbol.TakerFee.Equal(retrieved.TakerFee))
}
