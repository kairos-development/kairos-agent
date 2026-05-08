package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// StrategyRepository implements domain storage.StrategyRepository for SQLite.
type StrategyRepository struct {
	db *DB
}

// NewStrategyRepository creates a new SQLite strategy repository.
func NewStrategyRepository(db *DB) *StrategyRepository {
	return &StrategyRepository{db: db}
}

// Create persists a new strategy.
func (r *StrategyRepository) Create(ctx context.Context, strategy *entity.Strategy) error {
	query := `
		INSERT INTO strategies (
			id, name, type, status, symbol, max_position_size, max_daily_loss,
			current_pnl, daily_pnl, total_trades, winning_trades, losing_trades,
			created_at_utc, updated_at_utc, started_at_utc, stopped_at_utc
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := getTx(ctx, r.db.Conn())
	_, err := executor.ExecContext(ctx, query,
		strategy.ID,
		strategy.Name,
		strategy.Type,
		strategy.Status,
		strategy.Symbol,
		strategy.MaxPositionSize.String(),
		strategy.MaxDailyLoss.String(),
		strategy.CurrentPnL.String(),
		strategy.DailyPnL.String(),
		strategy.TotalTrades,
		strategy.WinningTrades,
		strategy.LosingTrades,
		strategy.CreatedAtUTC.Format(time.RFC3339Nano),
		strategy.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(strategy.StartedAtUTC),
		nullTime(strategy.StoppedAtUTC),
	)

	if err != nil {
		if isConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create strategy: %w", err)
	}

	return nil
}

// Update modifies an existing strategy.
func (r *StrategyRepository) Update(ctx context.Context, strategy *entity.Strategy) error {
	query := `
		UPDATE strategies SET
			name = ?,
			type = ?,
			status = ?,
			symbol = ?,
			max_position_size = ?,
			max_daily_loss = ?,
			current_pnl = ?,
			daily_pnl = ?,
			total_trades = ?,
			winning_trades = ?,
			losing_trades = ?,
			updated_at_utc = ?,
			started_at_utc = ?,
			stopped_at_utc = ?
		WHERE id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	result, err := executor.ExecContext(ctx, query,
		strategy.Name,
		strategy.Type,
		strategy.Status,
		strategy.Symbol,
		strategy.MaxPositionSize.String(),
		strategy.MaxDailyLoss.String(),
		strategy.CurrentPnL.String(),
		strategy.DailyPnL.String(),
		strategy.TotalTrades,
		strategy.WinningTrades,
		strategy.LosingTrades,
		strategy.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(strategy.StartedAtUTC),
		nullTime(strategy.StoppedAtUTC),
		strategy.ID,
	)

	if err != nil {
		return fmt.Errorf("update strategy: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetByID retrieves a strategy by its ID.
func (r *StrategyRepository) GetByID(ctx context.Context, id string) (*entity.Strategy, error) {
	query := `
		SELECT id, name, type, status, symbol, max_position_size, max_daily_loss,
			current_pnl, daily_pnl, total_trades, winning_trades, losing_trades,
			created_at_utc, updated_at_utc, started_at_utc, stopped_at_utc
		FROM strategies
		WHERE id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, id)

	strategy, err := scanStrategy(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get strategy by id: %w", err)
	}

	return strategy, nil
}

// List retrieves all strategies with pagination.
func (r *StrategyRepository) List(ctx context.Context, limit, offset int) ([]*entity.Strategy, error) {
	query := `
		SELECT id, name, type, status, symbol, max_position_size, max_daily_loss,
			current_pnl, daily_pnl, total_trades, winning_trades, losing_trades,
			created_at_utc, updated_at_utc, started_at_utc, stopped_at_utc
		FROM strategies
		ORDER BY created_at_utc DESC
		LIMIT ? OFFSET ?
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list strategies: %w", err)
	}
	defer rows.Close()

	return scanStrategies(rows)
}

// ListActive retrieves all active strategies.
func (r *StrategyRepository) ListActive(ctx context.Context) ([]*entity.Strategy, error) {
	query := `
		SELECT id, name, type, status, symbol, max_position_size, max_daily_loss,
			current_pnl, daily_pnl, total_trades, winning_trades, losing_trades,
			created_at_utc, updated_at_utc, started_at_utc, stopped_at_utc
		FROM strategies
		WHERE status = 'active'
		ORDER BY created_at_utc DESC
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active strategies: %w", err)
	}
	defer rows.Close()

	return scanStrategies(rows)
}

// Delete removes a strategy.
func (r *StrategyRepository) Delete(ctx context.Context, id string) error {
	executor := getTx(ctx, r.db.Conn())
	result, err := executor.ExecContext(ctx, "DELETE FROM strategies WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete strategy: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func scanStrategy(row interface {
	Scan(dest ...interface{}) error
}) (*entity.Strategy, error) {
	var (
		strategy                                          entity.Strategy
		maxPosStr, maxLossStr, currentPnLStr, dailyPnLStr string
		createdAt, updatedAt                              string
		startedAt, stoppedAt                              sql.NullString
	)

	err := row.Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.Type,
		&strategy.Status,
		&strategy.Symbol,
		&maxPosStr,
		&maxLossStr,
		&currentPnLStr,
		&dailyPnLStr,
		&strategy.TotalTrades,
		&strategy.WinningTrades,
		&strategy.LosingTrades,
		&createdAt,
		&updatedAt,
		&startedAt,
		&stoppedAt,
	)

	if err != nil {
		return nil, err
	}

	strategy.MaxPositionSize, _ = decimal.NewFromString(maxPosStr)
	strategy.MaxDailyLoss, _ = decimal.NewFromString(maxLossStr)
	strategy.CurrentPnL, _ = decimal.NewFromString(currentPnLStr)
	strategy.DailyPnL, _ = decimal.NewFromString(dailyPnLStr)

	strategy.CreatedAtUTC, _ = time.Parse(time.RFC3339Nano, createdAt)
	strategy.UpdatedAtUTC, _ = time.Parse(time.RFC3339Nano, updatedAt)

	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, startedAt.String)
		strategy.StartedAtUTC = &t
	}

	if stoppedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, stoppedAt.String)
		strategy.StoppedAtUTC = &t
	}

	return &strategy, nil
}

func scanStrategies(rows *sql.Rows) ([]*entity.Strategy, error) {
	var strategies []*entity.Strategy

	for rows.Next() {
		strategy, err := scanStrategy(rows)
		if err != nil {
			return nil, err
		}
		strategies = append(strategies, strategy)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return strategies, nil
}
