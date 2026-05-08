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

func setupPositionTestDB(t *testing.T) (*DB, func()) {
	tmpFile, err := os.CreateTemp("", "test_positions_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	db, err := Open(tmpFile.Name())
	require.NoError(t, err)

	// Create positions table
	_, err = db.Conn().Exec(`
		CREATE TABLE positions (
			id TEXT PRIMARY KEY,
			strategy_id TEXT NOT NULL,
			symbol TEXT NOT NULL,
			side TEXT NOT NULL,
			quantity TEXT NOT NULL,
			entry_price TEXT NOT NULL,
			current_price TEXT NOT NULL,
			unrealized_pnl TEXT NOT NULL,
			realized_pnl TEXT NOT NULL,
			opened_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			closed_at_utc TEXT
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestNewPositionRepository(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestPositionRepository_Create(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	position := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(500),
		RealizedPnL:   decimal.Zero,
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, position)
	assert.NoError(t, err)

	// Verify position was created
	retrieved, err := repo.GetByID(ctx, "pos1")
	require.NoError(t, err)
	assert.Equal(t, "pos1", retrieved.ID)
	assert.Equal(t, "BTCUSDT", retrieved.Symbol)
	assert.True(t, retrieved.Quantity.Equal(decimal.NewFromFloat(0.5)))
	assert.True(t, retrieved.EntryPrice.Equal(decimal.NewFromInt(50000)))
}

func TestPositionRepository_Create_DuplicateKey(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	position := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(500),
		RealizedPnL:   decimal.Zero,
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, position)
	require.NoError(t, err)

	// Try to create duplicate
	err = repo.Create(ctx, position)
	assert.ErrorIs(t, err, ErrDuplicateKey)
}

func TestPositionRepository_Update(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	position := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(50000),
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.Zero,
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, position)
	require.NoError(t, err)

	// Update position
	position.CurrentPrice = decimal.NewFromInt(51000)
	position.UnrealizedPnL = decimal.NewFromInt(500)
	position.UpdatedAtUTC = now.Add(1 * time.Second)

	err = repo.Update(ctx, position)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, "pos1")
	require.NoError(t, err)
	assert.True(t, retrieved.CurrentPrice.Equal(decimal.NewFromInt(51000)))
	assert.True(t, retrieved.UnrealizedPnL.Equal(decimal.NewFromInt(500)))
}

func TestPositionRepository_Update_NotFound(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	position := &entity.Position{
		ID:           "nonexistent",
		UpdatedAtUTC: now,
	}

	err := repo.Update(ctx, position)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPositionRepository_GetByID(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	position := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(500),
		RealizedPnL:   decimal.Zero,
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, position)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, "pos1")
	require.NoError(t, err)
	assert.Equal(t, "pos1", retrieved.ID)
	assert.Equal(t, "strategy1", retrieved.StrategyID)
}

func TestPositionRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPositionRepository_GetByStrategyAndSymbol(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	position := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(500),
		RealizedPnL:   decimal.Zero,
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
	}

	err := repo.Create(ctx, position)
	require.NoError(t, err)

	retrieved, err := repo.GetByStrategyAndSymbol(ctx, "strategy1", "BTCUSDT")
	require.NoError(t, err)
	assert.Equal(t, "pos1", retrieved.ID)
	assert.Equal(t, "strategy1", retrieved.StrategyID)
	assert.Equal(t, "BTCUSDT", retrieved.Symbol)
}

func TestPositionRepository_GetByStrategyAndSymbol_NotFound(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByStrategyAndSymbol(ctx, "strategy1", "BTCUSDT")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPositionRepository_GetByStrategyAndSymbol_OnlyOpen(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	closedAt := now.Add(1 * time.Hour)

	// Create closed position
	closedPosition := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.Zero,
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.NewFromInt(500),
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
		ClosedAtUTC:   &closedAt,
	}

	err := repo.Create(ctx, closedPosition)
	require.NoError(t, err)

	// Should not find closed position
	_, err = repo.GetByStrategyAndSymbol(ctx, "strategy1", "BTCUSDT")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPositionRepository_ListByStrategy(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create multiple positions for same strategy
	for i := 1; i <= 3; i++ {
		position := &entity.Position{
			ID:            "pos" + string(rune('0'+i)),
			StrategyID:    "strategy1",
			Symbol:        "BTCUSDT",
			Side:          entity.PositionSideLong,
			Quantity:      decimal.NewFromFloat(0.5),
			EntryPrice:    decimal.NewFromInt(50000),
			CurrentPrice:  decimal.NewFromInt(51000),
			UnrealizedPnL: decimal.NewFromInt(500),
			RealizedPnL:   decimal.Zero,
			OpenedAtUTC:   now.Add(time.Duration(i) * time.Second),
			UpdatedAtUTC:  now.Add(time.Duration(i) * time.Second),
		}
		err := repo.Create(ctx, position)
		require.NoError(t, err)
	}

	positions, err := repo.ListByStrategy(ctx, "strategy1")
	require.NoError(t, err)
	assert.Len(t, positions, 3)
}

func TestPositionRepository_ListOpen(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	closedAt := now.Add(1 * time.Hour)

	// Create open position
	openPosition := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.NewFromFloat(0.5),
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.NewFromInt(500),
		RealizedPnL:   decimal.Zero,
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
	}

	// Create closed position
	closedPosition := &entity.Position{
		ID:            "pos2",
		StrategyID:    "strategy1",
		Symbol:        "ETHUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.Zero,
		EntryPrice:    decimal.NewFromInt(3000),
		CurrentPrice:  decimal.NewFromInt(3100),
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.NewFromInt(100),
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
		ClosedAtUTC:   &closedAt,
	}

	err := repo.Create(ctx, openPosition)
	require.NoError(t, err)
	err = repo.Create(ctx, closedPosition)
	require.NoError(t, err)

	// ListOpen should return only open positions
	positions, err := repo.ListOpen(ctx)
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.Equal(t, "pos1", positions[0].ID)
	assert.Nil(t, positions[0].ClosedAtUTC)
}

func TestPositionRepository_ListOpen_Empty(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()

	positions, err := repo.ListOpen(ctx)
	require.NoError(t, err)
	assert.Empty(t, positions)
}

func TestPositionRepository_WithClosedPosition(t *testing.T) {
	db, cleanup := setupPositionTestDB(t)
	defer cleanup()

	repo := NewPositionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	closedAt := now.Add(1 * time.Hour)

	position := &entity.Position{
		ID:            "pos1",
		StrategyID:    "strategy1",
		Symbol:        "BTCUSDT",
		Side:          entity.PositionSideLong,
		Quantity:      decimal.Zero,
		EntryPrice:    decimal.NewFromInt(50000),
		CurrentPrice:  decimal.NewFromInt(51000),
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.NewFromInt(500),
		OpenedAtUTC:   now,
		UpdatedAtUTC:  now,
		ClosedAtUTC:   &closedAt,
	}

	err := repo.Create(ctx, position)
	require.NoError(t, err)

	// Verify closed position was stored correctly
	retrieved, err := repo.GetByID(ctx, "pos1")
	require.NoError(t, err)
	assert.NotNil(t, retrieved.ClosedAtUTC)
	assert.True(t, retrieved.ClosedAtUTC.Equal(closedAt))
}
