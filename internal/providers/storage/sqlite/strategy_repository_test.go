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

func setupStrategyTestDB(t *testing.T) (*DB, func()) {
	tmpFile, err := os.CreateTemp("", "test_strategies_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	db, err := Open(tmpFile.Name())
	require.NoError(t, err)

	// Create strategies table
	_, err = db.Conn().Exec(`
		CREATE TABLE strategies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			symbol TEXT NOT NULL,
			max_position_size TEXT NOT NULL,
			max_daily_loss TEXT NOT NULL,
			current_pnl TEXT NOT NULL,
			daily_pnl TEXT NOT NULL,
			total_trades INTEGER NOT NULL,
			winning_trades INTEGER NOT NULL,
			losing_trades INTEGER NOT NULL,
			created_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			started_at_utc TEXT,
			stopped_at_utc TEXT
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestNewStrategyRepository(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestStrategyRepository_Create(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	strategy := &entity.Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            entity.StrategyTypePaper,
		Status:          entity.StrategyStatusActive,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.Zero,
		DailyPnL:        decimal.Zero,
		TotalTrades:     0,
		WinningTrades:   0,
		LosingTrades:    0,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
	}

	err := repo.Create(ctx, strategy)
	assert.NoError(t, err)

	// Verify strategy was created
	retrieved, err := repo.GetByID(ctx, "strat1")
	require.NoError(t, err)
	assert.Equal(t, "strat1", retrieved.ID)
	assert.Equal(t, "SMA-Cross", retrieved.Name)
	assert.Equal(t, entity.StrategyTypePaper, retrieved.Type)
	assert.True(t, retrieved.MaxPositionSize.Equal(decimal.NewFromInt(1)))
}

func TestStrategyRepository_Create_DuplicateKey(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	strategy := &entity.Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            entity.StrategyTypePaper,
		Status:          entity.StrategyStatusActive,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.Zero,
		DailyPnL:        decimal.Zero,
		TotalTrades:     0,
		WinningTrades:   0,
		LosingTrades:    0,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
	}

	err := repo.Create(ctx, strategy)
	require.NoError(t, err)

	// Try to create duplicate
	err = repo.Create(ctx, strategy)
	assert.ErrorIs(t, err, ErrDuplicateKey)
}

func TestStrategyRepository_Update(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	strategy := &entity.Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            entity.StrategyTypePaper,
		Status:          entity.StrategyStatusIdle,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.Zero,
		DailyPnL:        decimal.Zero,
		TotalTrades:     0,
		WinningTrades:   0,
		LosingTrades:    0,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
	}

	err := repo.Create(ctx, strategy)
	require.NoError(t, err)

	// Update strategy
	strategy.Status = entity.StrategyStatusActive
	strategy.CurrentPnL = decimal.NewFromInt(500)
	strategy.TotalTrades = 10
	strategy.WinningTrades = 6
	strategy.UpdatedAtUTC = now.Add(1 * time.Second)
	startedAt := now.Add(1 * time.Second)
	strategy.StartedAtUTC = &startedAt

	err = repo.Update(ctx, strategy)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, "strat1")
	require.NoError(t, err)
	assert.Equal(t, entity.StrategyStatusActive, retrieved.Status)
	assert.True(t, retrieved.CurrentPnL.Equal(decimal.NewFromInt(500)))
	assert.Equal(t, 10, retrieved.TotalTrades)
	assert.Equal(t, 6, retrieved.WinningTrades)
	assert.NotNil(t, retrieved.StartedAtUTC)
}

func TestStrategyRepository_Update_NotFound(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	strategy := &entity.Strategy{
		ID:           "nonexistent",
		UpdatedAtUTC: now,
	}

	err := repo.Update(ctx, strategy)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStrategyRepository_GetByID(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	strategy := &entity.Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            entity.StrategyTypePaper,
		Status:          entity.StrategyStatusActive,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.Zero,
		DailyPnL:        decimal.Zero,
		TotalTrades:     0,
		WinningTrades:   0,
		LosingTrades:    0,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
	}

	err := repo.Create(ctx, strategy)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, "strat1")
	require.NoError(t, err)
	assert.Equal(t, "strat1", retrieved.ID)
	assert.Equal(t, "SMA-Cross", retrieved.Name)
}

func TestStrategyRepository_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStrategyRepository_List(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create multiple strategies
	for i := 1; i <= 3; i++ {
		strategy := &entity.Strategy{
			ID:              "strat" + string(rune('0'+i)),
			Name:            "Strategy-" + string(rune('0'+i)),
			Type:            entity.StrategyTypePaper,
			Status:          entity.StrategyStatusIdle,
			Symbol:          "BTCUSDT",
			MaxPositionSize: decimal.NewFromInt(1),
			MaxDailyLoss:    decimal.NewFromInt(1000),
			CurrentPnL:      decimal.Zero,
			DailyPnL:        decimal.Zero,
			TotalTrades:     0,
			WinningTrades:   0,
			LosingTrades:    0,
			CreatedAtUTC:    now.Add(time.Duration(i) * time.Second),
			UpdatedAtUTC:    now.Add(time.Duration(i) * time.Second),
		}
		err := repo.Create(ctx, strategy)
		require.NoError(t, err)
	}

	strategies, err := repo.List(ctx, 10, 0)
	require.NoError(t, err)
	assert.Len(t, strategies, 3)
}

func TestStrategyRepository_List_Pagination(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create 5 strategies
	for i := 1; i <= 5; i++ {
		strategy := &entity.Strategy{
			ID:              "strat" + string(rune('0'+i)),
			Name:            "Strategy-" + string(rune('0'+i)),
			Type:            entity.StrategyTypePaper,
			Status:          entity.StrategyStatusIdle,
			Symbol:          "BTCUSDT",
			MaxPositionSize: decimal.NewFromInt(1),
			MaxDailyLoss:    decimal.NewFromInt(1000),
			CurrentPnL:      decimal.Zero,
			DailyPnL:        decimal.Zero,
			TotalTrades:     0,
			WinningTrades:   0,
			LosingTrades:    0,
			CreatedAtUTC:    now.Add(time.Duration(i) * time.Second),
			UpdatedAtUTC:    now.Add(time.Duration(i) * time.Second),
		}
		err := repo.Create(ctx, strategy)
		require.NoError(t, err)
	}

	// Get first page
	page1, err := repo.List(ctx, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page
	page2, err := repo.List(ctx, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify different results
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestStrategyRepository_ListActive(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create strategies with different statuses
	statuses := []entity.StrategyStatus{
		entity.StrategyStatusIdle,
		entity.StrategyStatusActive,
		entity.StrategyStatusActive,
		entity.StrategyStatusPaused,
	}

	for i, status := range statuses {
		strategy := &entity.Strategy{
			ID:              "strat" + string(rune('0'+i+1)),
			Name:            "Strategy-" + string(rune('0'+i+1)),
			Type:            entity.StrategyTypePaper,
			Status:          status,
			Symbol:          "BTCUSDT",
			MaxPositionSize: decimal.NewFromInt(1),
			MaxDailyLoss:    decimal.NewFromInt(1000),
			CurrentPnL:      decimal.Zero,
			DailyPnL:        decimal.Zero,
			TotalTrades:     0,
			WinningTrades:   0,
			LosingTrades:    0,
			CreatedAtUTC:    now,
			UpdatedAtUTC:    now,
		}
		err := repo.Create(ctx, strategy)
		require.NoError(t, err)
	}

	// ListActive should return only active strategies
	strategies, err := repo.ListActive(ctx)
	require.NoError(t, err)
	assert.Len(t, strategies, 2)
	for _, strategy := range strategies {
		assert.Equal(t, entity.StrategyStatusActive, strategy.Status)
	}
}

func TestStrategyRepository_Delete(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	strategy := &entity.Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            entity.StrategyTypePaper,
		Status:          entity.StrategyStatusIdle,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.Zero,
		DailyPnL:        decimal.Zero,
		TotalTrades:     0,
		WinningTrades:   0,
		LosingTrades:    0,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
	}

	err := repo.Create(ctx, strategy)
	require.NoError(t, err)

	// Delete strategy
	err = repo.Delete(ctx, "strat1")
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, "strat1")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStrategyRepository_Delete_NotFound(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStrategyRepository_WithTimestamps(t *testing.T) {
	db, cleanup := setupStrategyTestDB(t)
	defer cleanup()

	repo := NewStrategyRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	startedAt := now.Add(1 * time.Hour)
	stoppedAt := now.Add(2 * time.Hour)

	strategy := &entity.Strategy{
		ID:              "strat1",
		Name:            "SMA-Cross",
		Type:            entity.StrategyTypePaper,
		Status:          entity.StrategyStatusStopped,
		Symbol:          "BTCUSDT",
		MaxPositionSize: decimal.NewFromInt(1),
		MaxDailyLoss:    decimal.NewFromInt(1000),
		CurrentPnL:      decimal.NewFromInt(500),
		DailyPnL:        decimal.NewFromInt(100),
		TotalTrades:     10,
		WinningTrades:   6,
		LosingTrades:    4,
		CreatedAtUTC:    now,
		UpdatedAtUTC:    now,
		StartedAtUTC:    &startedAt,
		StoppedAtUTC:    &stoppedAt,
	}

	err := repo.Create(ctx, strategy)
	require.NoError(t, err)

	// Verify timestamps were stored correctly
	retrieved, err := repo.GetByID(ctx, "strat1")
	require.NoError(t, err)
	assert.NotNil(t, retrieved.StartedAtUTC)
	assert.NotNil(t, retrieved.StoppedAtUTC)
	assert.True(t, retrieved.StartedAtUTC.Equal(startedAt))
	assert.True(t, retrieved.StoppedAtUTC.Equal(stoppedAt))
}
