package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

// PositionRepository implements domain storage.PositionRepository for SQLite.
type PositionRepository struct {
	db *DB
}

// NewPositionRepository creates a new SQLite position repository.
func NewPositionRepository(db *DB) *PositionRepository {
	return &PositionRepository{db: db}
}

// Create persists a new position.
func (r *PositionRepository) Create(ctx context.Context, position *entity.Position) error {
	query := `
		INSERT INTO positions (
			id, strategy_id, symbol, side, quantity, entry_price,
			current_price, unrealized_pnl, realized_pnl,
			opened_at_utc, updated_at_utc, closed_at_utc
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := getTx(ctx, r.db.Conn())
	_, err := executor.ExecContext(ctx, query,
		position.ID,
		position.StrategyID,
		position.Symbol,
		position.Side,
		position.Quantity.String(),
		position.EntryPrice.String(),
		position.CurrentPrice.String(),
		position.UnrealizedPnL.String(),
		position.RealizedPnL.String(),
		position.OpenedAtUTC.Format(time.RFC3339Nano),
		position.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(position.ClosedAtUTC),
	)

	if err != nil {
		if isConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create position: %w", err)
	}

	return nil
}

// Update modifies an existing position.
func (r *PositionRepository) Update(ctx context.Context, position *entity.Position) error {
	query := `
		UPDATE positions SET
			side = ?,
			quantity = ?,
			entry_price = ?,
			current_price = ?,
			unrealized_pnl = ?,
			realized_pnl = ?,
			updated_at_utc = ?,
			closed_at_utc = ?
		WHERE id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	result, err := executor.ExecContext(ctx, query,
		position.Side,
		position.Quantity.String(),
		position.EntryPrice.String(),
		position.CurrentPrice.String(),
		position.UnrealizedPnL.String(),
		position.RealizedPnL.String(),
		position.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(position.ClosedAtUTC),
		position.ID,
	)

	if err != nil {
		return fmt.Errorf("update position: %w", err)
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

// GetByID retrieves a position by its ID.
func (r *PositionRepository) GetByID(ctx context.Context, id string) (*entity.Position, error) {
	query := `
		SELECT id, strategy_id, symbol, side, quantity, entry_price,
			current_price, unrealized_pnl, realized_pnl,
			opened_at_utc, updated_at_utc, closed_at_utc
		FROM positions
		WHERE id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, id)

	position, err := scanPosition(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get position by id: %w", err)
	}

	return position, nil
}

// GetByStrategyAndSymbol retrieves the current position for a strategy and symbol.
func (r *PositionRepository) GetByStrategyAndSymbol(ctx context.Context, strategyID, symbol string) (*entity.Position, error) {
	query := `
		SELECT id, strategy_id, symbol, side, quantity, entry_price,
			current_price, unrealized_pnl, realized_pnl,
			opened_at_utc, updated_at_utc, closed_at_utc
		FROM positions
		WHERE strategy_id = ? AND symbol = ? AND closed_at_utc IS NULL
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, strategyID, symbol)

	position, err := scanPosition(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get position by strategy and symbol: %w", err)
	}

	return position, nil
}

// ListByStrategy retrieves all positions for a strategy.
func (r *PositionRepository) ListByStrategy(ctx context.Context, strategyID string) ([]*entity.Position, error) {
	query := `
		SELECT id, strategy_id, symbol, side, quantity, entry_price,
			current_price, unrealized_pnl, realized_pnl,
			opened_at_utc, updated_at_utc, closed_at_utc
		FROM positions
		WHERE strategy_id = ?
		ORDER BY opened_at_utc DESC
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query, strategyID)
	if err != nil {
		return nil, fmt.Errorf("list positions by strategy: %w", err)
	}
	defer rows.Close()

	return scanPositions(rows)
}

// ListOpen retrieves all open positions across all strategies.
func (r *PositionRepository) ListOpen(ctx context.Context) ([]*entity.Position, error) {
	query := `
		SELECT id, strategy_id, symbol, side, quantity, entry_price,
			current_price, unrealized_pnl, realized_pnl,
			opened_at_utc, updated_at_utc, closed_at_utc
		FROM positions
		WHERE closed_at_utc IS NULL
		ORDER BY opened_at_utc DESC
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list open positions: %w", err)
	}
	defer rows.Close()

	return scanPositions(rows)
}

func scanPosition(row interface {
	Scan(dest ...interface{}) error
}) (*entity.Position, error) {
	var (
		position                                                           entity.Position
		qtyStr, entryPriceStr, currentPriceStr, unrealizedStr, realizedStr string
		openedAt, updatedAt                                                string
		closedAt                                                           sql.NullString
	)

	err := row.Scan(
		&position.ID,
		&position.StrategyID,
		&position.Symbol,
		&position.Side,
		&qtyStr,
		&entryPriceStr,
		&currentPriceStr,
		&unrealizedStr,
		&realizedStr,
		&openedAt,
		&updatedAt,
		&closedAt,
	)

	if err != nil {
		return nil, err
	}

	if position.Quantity, err = parseDecimalField("positions.quantity", qtyStr); err != nil {
		return nil, err
	}
	if position.EntryPrice, err = parseDecimalField("positions.entry_price", entryPriceStr); err != nil {
		return nil, err
	}
	if position.CurrentPrice, err = parseDecimalField("positions.current_price", currentPriceStr); err != nil {
		return nil, err
	}
	if position.UnrealizedPnL, err = parseDecimalField("positions.unrealized_pnl", unrealizedStr); err != nil {
		return nil, err
	}
	if position.RealizedPnL, err = parseDecimalField("positions.realized_pnl", realizedStr); err != nil {
		return nil, err
	}

	if position.OpenedAtUTC, err = parseTimeField("positions.opened_at_utc", openedAt); err != nil {
		return nil, err
	}
	if position.UpdatedAtUTC, err = parseTimeField("positions.updated_at_utc", updatedAt); err != nil {
		return nil, err
	}

	if closedAt.Valid {
		t, err := parseTimeField("positions.closed_at_utc", closedAt.String)
		if err != nil {
			return nil, err
		}
		position.ClosedAtUTC = &t
	}

	return &position, nil
}

func scanPositions(rows *sql.Rows) ([]*entity.Position, error) {
	var positions []*entity.Position

	for rows.Next() {
		position, err := scanPosition(rows)
		if err != nil {
			return nil, err
		}
		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return positions, nil
}
