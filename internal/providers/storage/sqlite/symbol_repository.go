package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// SymbolRepository implements domain storage.SymbolRepository for SQLite.
type SymbolRepository struct {
	db *DB
}

// NewSymbolRepository creates a new SQLite symbol repository.
func NewSymbolRepository(db *DB) *SymbolRepository {
	return &SymbolRepository{db: db}
}

// Upsert creates or updates symbol metadata.
func (r *SymbolRepository) Upsert(ctx context.Context, symbol *entity.Symbol) error {
	query := `
		INSERT INTO symbols (
			name, base_currency, quote_currency, status,
			min_order_qty, max_order_qty, min_price, max_price,
			tick_size, step_size, min_notional, maker_fee, taker_fee,
			updated_at_utc
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			base_currency = excluded.base_currency,
			quote_currency = excluded.quote_currency,
			status = excluded.status,
			min_order_qty = excluded.min_order_qty,
			max_order_qty = excluded.max_order_qty,
			min_price = excluded.min_price,
			max_price = excluded.max_price,
			tick_size = excluded.tick_size,
			step_size = excluded.step_size,
			min_notional = excluded.min_notional,
			maker_fee = excluded.maker_fee,
			taker_fee = excluded.taker_fee,
			updated_at_utc = excluded.updated_at_utc
	`

	executor := getTx(ctx, r.db.Conn())
	_, err := executor.ExecContext(ctx, query,
		symbol.Name,
		symbol.BaseCurrency,
		symbol.QuoteCurrency,
		symbol.Status,
		symbol.MinOrderQty.String(),
		symbol.MaxOrderQty.String(),
		symbol.MinPrice.String(),
		symbol.MaxPrice.String(),
		symbol.TickSize.String(),
		symbol.StepSize.String(),
		symbol.MinNotional.String(),
		symbol.MakerFee.String(),
		symbol.TakerFee.String(),
		symbol.UpdatedAtUTC.Format(time.RFC3339Nano),
	)

	if err != nil {
		return fmt.Errorf("upsert symbol: %w", err)
	}

	return nil
}

// GetByName retrieves symbol metadata by name.
func (r *SymbolRepository) GetByName(ctx context.Context, name string) (*entity.Symbol, error) {
	query := `
		SELECT name, base_currency, quote_currency, status,
			min_order_qty, max_order_qty, min_price, max_price,
			tick_size, step_size, min_notional, maker_fee, taker_fee,
			updated_at_utc
		FROM symbols
		WHERE name = ?
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, name)

	symbol, err := scanSymbol(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get symbol by name: %w", err)
	}

	return symbol, nil
}

// List retrieves all symbols.
func (r *SymbolRepository) List(ctx context.Context) ([]*entity.Symbol, error) {
	query := `
		SELECT name, base_currency, quote_currency, status,
			min_order_qty, max_order_qty, min_price, max_price,
			tick_size, step_size, min_notional, maker_fee, taker_fee,
			updated_at_utc
		FROM symbols
		ORDER BY name
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list symbols: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}

// ListByStatus retrieves symbols filtered by status.
func (r *SymbolRepository) ListByStatus(ctx context.Context, status entity.SymbolStatus) ([]*entity.Symbol, error) {
	query := `
		SELECT name, base_currency, quote_currency, status,
			min_order_qty, max_order_qty, min_price, max_price,
			tick_size, step_size, min_notional, maker_fee, taker_fee,
			updated_at_utc
		FROM symbols
		WHERE status = ?
		ORDER BY name
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("list symbols by status: %w", err)
	}
	defer rows.Close()

	return scanSymbols(rows)
}

func scanSymbol(row interface {
	Scan(dest ...interface{}) error
}) (*entity.Symbol, error) {
	var (
		symbol                                                             entity.Symbol
		minQtyStr, maxQtyStr, minPriceStr, maxPriceStr                     string
		tickSizeStr, stepSizeStr, minNotionalStr, makerFeeStr, takerFeeStr string
		updatedAt                                                          string
	)

	err := row.Scan(
		&symbol.Name,
		&symbol.BaseCurrency,
		&symbol.QuoteCurrency,
		&symbol.Status,
		&minQtyStr,
		&maxQtyStr,
		&minPriceStr,
		&maxPriceStr,
		&tickSizeStr,
		&stepSizeStr,
		&minNotionalStr,
		&makerFeeStr,
		&takerFeeStr,
		&updatedAt,
	)

	if err != nil {
		return nil, err
	}

	symbol.MinOrderQty, _ = decimal.NewFromString(minQtyStr)
	symbol.MaxOrderQty, _ = decimal.NewFromString(maxQtyStr)
	symbol.MinPrice, _ = decimal.NewFromString(minPriceStr)
	symbol.MaxPrice, _ = decimal.NewFromString(maxPriceStr)
	symbol.TickSize, _ = decimal.NewFromString(tickSizeStr)
	symbol.StepSize, _ = decimal.NewFromString(stepSizeStr)
	symbol.MinNotional, _ = decimal.NewFromString(minNotionalStr)
	symbol.MakerFee, _ = decimal.NewFromString(makerFeeStr)
	symbol.TakerFee, _ = decimal.NewFromString(takerFeeStr)
	symbol.UpdatedAtUTC, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &symbol, nil
}

func scanSymbols(rows *sql.Rows) ([]*entity.Symbol, error) {
	var symbols []*entity.Symbol

	for rows.Next() {
		symbol, err := scanSymbol(rows)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return symbols, nil
}
