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

func setupBalanceTestDB(t *testing.T) (*DB, func()) {
	tmpFile, err := os.CreateTemp("", "test_balances_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	db, err := Open(tmpFile.Name())
	require.NoError(t, err)

	// Create balances table
	_, err = db.Conn().Exec(`
		CREATE TABLE balances (
			asset TEXT NOT NULL,
			total TEXT NOT NULL,
			available TEXT NOT NULL,
			locked TEXT NOT NULL,
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

func TestNewBalanceRepository(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestBalanceRepository_Save(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	accountBalance := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:        "USDT",
				Total:        decimal.NewFromInt(10000),
				Available:    decimal.NewFromInt(9500),
				Locked:       decimal.NewFromInt(500),
				UpdatedAtUTC: now,
			},
			{
				Asset:        "BTC",
				Total:        decimal.NewFromFloat(0.5),
				Available:    decimal.NewFromFloat(0.4),
				Locked:       decimal.NewFromFloat(0.1),
				UpdatedAtUTC: now,
			},
		},
		UpdatedAtUTC: now,
	}

	err := repo.Save(ctx, accountBalance)
	assert.NoError(t, err)

	// Verify balances were saved
	retrieved, err := repo.GetLatest(ctx)
	require.NoError(t, err)
	assert.Len(t, retrieved.Balances, 2)
	assert.Equal(t, "BTC", retrieved.Balances[0].Asset) // Ordered by asset
	assert.Equal(t, "USDT", retrieved.Balances[1].Asset)
}

func TestBalanceRepository_GetLatest_RejectsCorruptDecimal(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	_, err := db.Conn().Exec(
		`INSERT INTO balances (asset, total, available, locked, updated_at_utc) VALUES (?, ?, ?, ?, ?)`,
		"USDT", "not-a-decimal", "1", "0", time.Now().UTC().Format(time.RFC3339Nano),
	)
	require.NoError(t, err)

	_, err = NewBalanceRepository(db).GetLatest(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "balances.total")
}

func TestBalanceRepository_Save_EmptyBalances(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	accountBalance := &entity.AccountBalance{
		Balances:     []entity.Balance{},
		UpdatedAtUTC: now,
	}

	err := repo.Save(ctx, accountBalance)
	assert.NoError(t, err)
}

func TestBalanceRepository_GetLatest(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Save first snapshot
	snapshot1 := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:        "USDT",
				Total:        decimal.NewFromInt(10000),
				Available:    decimal.NewFromInt(9500),
				Locked:       decimal.NewFromInt(500),
				UpdatedAtUTC: now,
			},
		},
		UpdatedAtUTC: now,
	}

	err := repo.Save(ctx, snapshot1)
	require.NoError(t, err)

	// Save second snapshot (later)
	later := now.Add(1 * time.Hour)
	snapshot2 := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:        "USDT",
				Total:        decimal.NewFromInt(11000),
				Available:    decimal.NewFromInt(10500),
				Locked:       decimal.NewFromInt(500),
				UpdatedAtUTC: later,
			},
		},
		UpdatedAtUTC: later,
	}

	err = repo.Save(ctx, snapshot2)
	require.NoError(t, err)

	// GetLatest should return the second snapshot
	latest, err := repo.GetLatest(ctx)
	require.NoError(t, err)
	assert.Len(t, latest.Balances, 1)
	assert.True(t, latest.Balances[0].Total.Equal(decimal.NewFromInt(11000)))
}

func TestBalanceRepository_GetLatest_NotFound(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()

	_, err := repo.GetLatest(ctx)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBalanceRepository_GetHistory(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Save multiple snapshots
	for i := 0; i < 3; i++ {
		timestamp := now.Add(time.Duration(i) * time.Hour)
		snapshot := &entity.AccountBalance{
			Balances: []entity.Balance{
				{
					Asset:        "USDT",
					Total:        decimal.NewFromInt(10000 + int64(i*1000)),
					Available:    decimal.NewFromInt(9500 + int64(i*1000)),
					Locked:       decimal.NewFromInt(500),
					UpdatedAtUTC: timestamp,
				},
			},
			UpdatedAtUTC: timestamp,
		}
		err := repo.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get history
	from := now.Add(-1 * time.Hour)
	to := now.Add(5 * time.Hour)
	history, err := repo.GetHistory(ctx, from, to)
	require.NoError(t, err)
	assert.Len(t, history, 3)

	// Verify order (DESC)
	assert.True(t, history[0].Balances[0].Total.Equal(decimal.NewFromInt(12000)))
	assert.True(t, history[1].Balances[0].Total.Equal(decimal.NewFromInt(11000)))
	assert.True(t, history[2].Balances[0].Total.Equal(decimal.NewFromInt(10000)))
}

func TestBalanceRepository_GetHistory_EmptyRange(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Save snapshot
	snapshot := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:        "USDT",
				Total:        decimal.NewFromInt(10000),
				Available:    decimal.NewFromInt(9500),
				Locked:       decimal.NewFromInt(500),
				UpdatedAtUTC: now,
			},
		},
		UpdatedAtUTC: now,
	}
	err := repo.Save(ctx, snapshot)
	require.NoError(t, err)

	// Query range that doesn't include the snapshot
	from := now.Add(-2 * time.Hour)
	to := now.Add(-1 * time.Hour)
	history, err := repo.GetHistory(ctx, from, to)
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestBalanceRepository_GetHistory_PartialRange(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Save 5 snapshots
	for i := 0; i < 5; i++ {
		timestamp := now.Add(time.Duration(i) * time.Hour)
		snapshot := &entity.AccountBalance{
			Balances: []entity.Balance{
				{
					Asset:        "USDT",
					Total:        decimal.NewFromInt(10000 + int64(i*1000)),
					Available:    decimal.NewFromInt(9500 + int64(i*1000)),
					Locked:       decimal.NewFromInt(500),
					UpdatedAtUTC: timestamp,
				},
			},
			UpdatedAtUTC: timestamp,
		}
		err := repo.Save(ctx, snapshot)
		require.NoError(t, err)
	}

	// Get only middle 3 snapshots
	from := now.Add(1 * time.Hour)
	to := now.Add(3 * time.Hour)
	history, err := repo.GetHistory(ctx, from, to)
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestBalanceRepository_MultipleAssets(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	accountBalance := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:        "USDT",
				Total:        decimal.NewFromInt(10000),
				Available:    decimal.NewFromInt(9500),
				Locked:       decimal.NewFromInt(500),
				UpdatedAtUTC: now,
			},
			{
				Asset:        "BTC",
				Total:        decimal.NewFromFloat(0.5),
				Available:    decimal.NewFromFloat(0.4),
				Locked:       decimal.NewFromFloat(0.1),
				UpdatedAtUTC: now,
			},
			{
				Asset:        "ETH",
				Total:        decimal.NewFromInt(5),
				Available:    decimal.NewFromInt(4),
				Locked:       decimal.NewFromInt(1),
				UpdatedAtUTC: now,
			},
		},
		UpdatedAtUTC: now,
	}

	err := repo.Save(ctx, accountBalance)
	require.NoError(t, err)

	// Verify all assets were saved and ordered correctly
	retrieved, err := repo.GetLatest(ctx)
	require.NoError(t, err)
	assert.Len(t, retrieved.Balances, 3)
	assert.Equal(t, "BTC", retrieved.Balances[0].Asset)
	assert.Equal(t, "ETH", retrieved.Balances[1].Asset)
	assert.Equal(t, "USDT", retrieved.Balances[2].Asset)
}

func TestBalanceRepository_DecimalPrecision(t *testing.T) {
	db, cleanup := setupBalanceTestDB(t)
	defer cleanup()

	repo := NewBalanceRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	// Use high precision decimal values
	total, _ := decimal.NewFromString("0.123456789012345678")
	available, _ := decimal.NewFromString("0.100000000000000001")
	locked, _ := decimal.NewFromString("0.023456789012345677")

	accountBalance := &entity.AccountBalance{
		Balances: []entity.Balance{
			{
				Asset:        "BTC",
				Total:        total,
				Available:    available,
				Locked:       locked,
				UpdatedAtUTC: now,
			},
		},
		UpdatedAtUTC: now,
	}

	err := repo.Save(ctx, accountBalance)
	require.NoError(t, err)

	// Verify precision is preserved
	retrieved, err := repo.GetLatest(ctx)
	require.NoError(t, err)
	assert.Len(t, retrieved.Balances, 1)

	expected, _ := decimal.NewFromString("0.123456789012345678")
	assert.True(t, retrieved.Balances[0].Total.Equal(expected))
}
