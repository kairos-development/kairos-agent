package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// BalanceRepository implements domain storage.BalanceRepository for SQLite.
type BalanceRepository struct {
	db *DB
}

// NewBalanceRepository creates a new SQLite balance repository.
func NewBalanceRepository(db *DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// Save persists a balance snapshot.
func (r *BalanceRepository) Save(ctx context.Context, balance *entity.AccountBalance) error {
	tx, err := r.db.Conn().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, bal := range balance.Balances {
		query := `
			INSERT INTO balances (asset, total, available, locked, updated_at_utc)
			VALUES (?, ?, ?, ?, ?)
		`

		_, err := tx.ExecContext(ctx, query,
			bal.Asset,
			bal.Total.String(),
			bal.Available.String(),
			bal.Locked.String(),
			bal.UpdatedAtUTC.Format(time.RFC3339Nano),
		)

		if err != nil {
			return fmt.Errorf("save balance for %s: %w", bal.Asset, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetLatest retrieves the most recent balance snapshot.
func (r *BalanceRepository) GetLatest(ctx context.Context) (*entity.AccountBalance, error) {
	query := `
		SELECT asset, total, available, locked, updated_at_utc
		FROM balances
		WHERE updated_at_utc = (SELECT MAX(updated_at_utc) FROM balances)
		ORDER BY asset
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get latest balance: %w", err)
	}
	defer rows.Close()

	var balances []entity.Balance
	var latestUpdate time.Time

	for rows.Next() {
		var (
			bal                               entity.Balance
			totalStr, availableStr, lockedStr string
			updatedAt                         string
		)

		err := rows.Scan(&bal.Asset, &totalStr, &availableStr, &lockedStr, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}

		bal.Total, _ = decimal.NewFromString(totalStr)
		bal.Available, _ = decimal.NewFromString(availableStr)
		bal.Locked, _ = decimal.NewFromString(lockedStr)
		bal.UpdatedAtUTC, _ = time.Parse(time.RFC3339Nano, updatedAt)

		if latestUpdate.IsZero() {
			latestUpdate = bal.UpdatedAtUTC
		}

		balances = append(balances, bal)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balances: %w", err)
	}

	if len(balances) == 0 {
		return nil, ErrNotFound
	}

	return &entity.AccountBalance{
		Balances:     balances,
		UpdatedAtUTC: latestUpdate,
	}, nil
}

// GetHistory retrieves balance snapshots within a time range.
func (r *BalanceRepository) GetHistory(ctx context.Context, from, to time.Time) ([]*entity.AccountBalance, error) {
	query := `
		SELECT DISTINCT updated_at_utc
		FROM balances
		WHERE updated_at_utc >= ? AND updated_at_utc <= ?
		ORDER BY updated_at_utc DESC
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query,
		from.Format(time.RFC3339Nano),
		to.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("get balance history: %w", err)
	}
	defer rows.Close()

	var timestamps []time.Time
	for rows.Next() {
		var ts string
		if err := rows.Scan(&ts); err != nil {
			return nil, fmt.Errorf("scan timestamp: %w", err)
		}
		t, _ := time.Parse(time.RFC3339Nano, ts)
		timestamps = append(timestamps, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate timestamps: %w", err)
	}

	var snapshots []*entity.AccountBalance
	for _, ts := range timestamps {
		snapshot, err := r.getSnapshotAt(ctx, ts)
		if err != nil {
			return nil, fmt.Errorf("get snapshot at %v: %w", ts, err)
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func (r *BalanceRepository) getSnapshotAt(ctx context.Context, timestamp time.Time) (*entity.AccountBalance, error) {
	query := `
		SELECT asset, total, available, locked, updated_at_utc
		FROM balances
		WHERE updated_at_utc = ?
		ORDER BY asset
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query, timestamp.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	defer rows.Close()

	var balances []entity.Balance

	for rows.Next() {
		var (
			bal                               entity.Balance
			totalStr, availableStr, lockedStr string
			updatedAt                         string
		)

		err := rows.Scan(&bal.Asset, &totalStr, &availableStr, &lockedStr, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}

		bal.Total, _ = decimal.NewFromString(totalStr)
		bal.Available, _ = decimal.NewFromString(availableStr)
		bal.Locked, _ = decimal.NewFromString(lockedStr)
		bal.UpdatedAtUTC, _ = time.Parse(time.RFC3339Nano, updatedAt)

		balances = append(balances, bal)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate balances: %w", err)
	}

	return &entity.AccountBalance{
		Balances:     balances,
		UpdatedAtUTC: timestamp,
	}, nil
}
