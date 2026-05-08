package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// OrderRepository implements domain storage.OrderRepository for SQLite.
type OrderRepository struct {
	db *DB
}

// NewOrderRepository creates a new SQLite order repository.
func NewOrderRepository(db *DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create persists a new order to the critical stream.
func (r *OrderRepository) Create(ctx context.Context, order *entity.Order) error {
	query := `
		INSERT INTO orders (
			id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := getTx(ctx, r.db.Conn())
	_, err := executor.ExecContext(ctx, query,
		order.ID,
		order.ClientOrderID,
		nullString(order.ExchangeOrderID),
		order.StrategyID,
		order.Symbol,
		order.Side,
		order.Type,
		order.Status,
		order.TimeInForce,
		order.Quantity.String(),
		order.Price.String(),
		order.FilledQty.String(),
		order.RemainingQty.String(),
		order.AvgFillPrice.String(),
		order.CreatedAtUTC.Format(time.RFC3339Nano),
		order.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(order.SubmittedAtUTC),
		nullTime(order.FilledAtUTC),
	)

	if err != nil {
		if isConstraintError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("create order: %w", err)
	}

	return nil
}

// Update modifies an existing order state.
func (r *OrderRepository) Update(ctx context.Context, order *entity.Order) error {
	query := `
		UPDATE orders SET
			exchange_order_id = ?,
			status = ?,
			filled_qty = ?,
			remaining_qty = ?,
			avg_fill_price = ?,
			updated_at_utc = ?,
			submitted_at_utc = ?,
			filled_at_utc = ?
		WHERE id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	result, err := executor.ExecContext(ctx, query,
		nullString(order.ExchangeOrderID),
		order.Status,
		order.FilledQty.String(),
		order.RemainingQty.String(),
		order.AvgFillPrice.String(),
		order.UpdatedAtUTC.Format(time.RFC3339Nano),
		nullTime(order.SubmittedAtUTC),
		nullTime(order.FilledAtUTC),
		order.ID,
	)

	if err != nil {
		return fmt.Errorf("update order: %w", err)
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

// GetByID retrieves an order by its internal ID.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*entity.Order, error) {
	query := `
		SELECT id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		FROM orders
		WHERE id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, id)

	order, err := scanOrder(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get order by id: %w", err)
	}

	return order, nil
}

// GetByClientOrderID retrieves an order by client order ID.
func (r *OrderRepository) GetByClientOrderID(ctx context.Context, clientOrderID string) (*entity.Order, error) {
	query := `
		SELECT id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		FROM orders
		WHERE client_order_id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, clientOrderID)

	order, err := scanOrder(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get order by client_order_id: %w", err)
	}

	return order, nil
}

// GetByExchangeOrderID retrieves an order by exchange order ID.
func (r *OrderRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	query := `
		SELECT id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		FROM orders
		WHERE exchange_order_id = ?
	`

	executor := getTx(ctx, r.db.Conn())
	row := executor.QueryRowContext(ctx, query, exchangeOrderID)

	order, err := scanOrder(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get order by exchange_order_id: %w", err)
	}

	return order, nil
}

// ListByStrategy retrieves all orders for a strategy.
func (r *OrderRepository) ListByStrategy(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	query := `
		SELECT id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		FROM orders
		WHERE strategy_id = ?
		ORDER BY created_at_utc DESC
		LIMIT ? OFFSET ?
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query, strategyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders by strategy: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

// ListActive retrieves all active (submitted or partial) orders.
func (r *OrderRepository) ListActive(ctx context.Context) ([]*entity.Order, error) {
	query := `
		SELECT id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		FROM orders
		WHERE status IN ('submitted', 'partial')
		ORDER BY created_at_utc DESC
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active orders: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

// ListInFlight retrieves orders that may be in-flight after a crash.
func (r *OrderRepository) ListInFlight(ctx context.Context, since time.Time) ([]*entity.Order, error) {
	query := `
		SELECT id, client_order_id, exchange_order_id, strategy_id, symbol,
			side, type, status, time_in_force, quantity, price,
			filled_qty, remaining_qty, avg_fill_price,
			created_at_utc, updated_at_utc, submitted_at_utc, filled_at_utc
		FROM orders
		WHERE status IN ('pending', 'submitted', 'partial')
			AND created_at_utc >= ?
		ORDER BY created_at_utc ASC
	`

	executor := getTx(ctx, r.db.Conn())
	rows, err := executor.QueryContext(ctx, query, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("list in-flight orders: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

func scanOrder(row interface {
	Scan(dest ...interface{}) error
}) (*entity.Order, error) {
	var (
		order                                                        entity.Order
		exchangeOrderID, submittedAt, filledAt                       sql.NullString
		qtyStr, priceStr, filledQtyStr, remainingQtyStr, avgPriceStr string
		createdAt, updatedAt                                         string
	)

	err := row.Scan(
		&order.ID,
		&order.ClientOrderID,
		&exchangeOrderID,
		&order.StrategyID,
		&order.Symbol,
		&order.Side,
		&order.Type,
		&order.Status,
		&order.TimeInForce,
		&qtyStr,
		&priceStr,
		&filledQtyStr,
		&remainingQtyStr,
		&avgPriceStr,
		&createdAt,
		&updatedAt,
		&submittedAt,
		&filledAt,
	)

	if err != nil {
		return nil, err
	}

	if exchangeOrderID.Valid {
		order.ExchangeOrderID = exchangeOrderID.String
	}

	order.Quantity, _ = decimal.NewFromString(qtyStr)
	order.Price, _ = decimal.NewFromString(priceStr)
	order.FilledQty, _ = decimal.NewFromString(filledQtyStr)
	order.RemainingQty, _ = decimal.NewFromString(remainingQtyStr)
	order.AvgFillPrice, _ = decimal.NewFromString(avgPriceStr)

	order.CreatedAtUTC, _ = time.Parse(time.RFC3339Nano, createdAt)
	order.UpdatedAtUTC, _ = time.Parse(time.RFC3339Nano, updatedAt)

	if submittedAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, submittedAt.String)
		order.SubmittedAtUTC = &t
	}

	if filledAt.Valid {
		t, _ := time.Parse(time.RFC3339Nano, filledAt.String)
		order.FilledAtUTC = &t
	}

	return &order, nil
}

func scanOrders(rows *sql.Rows) ([]*entity.Order, error) {
	var orders []*entity.Order

	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: t.Format(time.RFC3339Nano), Valid: true}
}
