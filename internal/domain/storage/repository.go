package storage

import (
	"context"
	"errors"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

var (
	// ErrNotFound reports that the requested entity was not found.
	ErrNotFound = errors.New("entity not found")
)

// OrderRepository defines persistence operations for orders.
// Implementations must use UTC timestamps and enforce append-only semantics for critical stream.
type OrderRepository interface {
	// Create persists a new order to the critical stream.
	// Must use PRAGMA synchronous=FULL and explicit fsync.
	Create(ctx context.Context, order *entity.Order) error

	// Update modifies an existing order state.
	// Used for status transitions, partial fills, and final states.
	Update(ctx context.Context, order *entity.Order) error

	// GetByID retrieves an order by its internal ID.
	GetByID(ctx context.Context, id string) (*entity.Order, error)

	// GetByClientOrderID retrieves an order by client order ID.
	GetByClientOrderID(ctx context.Context, clientOrderID string) (*entity.Order, error)

	// GetByExchangeOrderID retrieves an order by exchange order ID.
	GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*entity.Order, error)

	// ListByStrategy retrieves all orders for a strategy.
	ListByStrategy(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error)

	// ListActive retrieves all active (submitted or partial) orders.
	ListActive(ctx context.Context) ([]*entity.Order, error)

	// ListInFlight retrieves orders that may be in-flight after a crash.
	// Used during recovery to identify orders requiring reconciliation.
	ListInFlight(ctx context.Context, since time.Time) ([]*entity.Order, error)
}

// PositionRepository defines persistence operations for positions.
type PositionRepository interface {
	// Create persists a new position.
	Create(ctx context.Context, position *entity.Position) error

	// Update modifies an existing position.
	Update(ctx context.Context, position *entity.Position) error

	// GetByID retrieves a position by its ID.
	GetByID(ctx context.Context, id string) (*entity.Position, error)

	// GetByStrategyAndSymbol retrieves the current position for a strategy and symbol.
	GetByStrategyAndSymbol(ctx context.Context, strategyID, symbol string) (*entity.Position, error)

	// ListByStrategy retrieves all positions for a strategy.
	ListByStrategy(ctx context.Context, strategyID string) ([]*entity.Position, error)

	// ListOpen retrieves all open positions across all strategies.
	ListOpen(ctx context.Context) ([]*entity.Position, error)
}

// StrategyRepository defines persistence operations for strategies.
type StrategyRepository interface {
	// Create persists a new strategy.
	Create(ctx context.Context, strategy *entity.Strategy) error

	// Update modifies an existing strategy.
	Update(ctx context.Context, strategy *entity.Strategy) error

	// GetByID retrieves a strategy by its ID.
	GetByID(ctx context.Context, id string) (*entity.Strategy, error)

	// List retrieves all strategies with pagination.
	List(ctx context.Context, limit, offset int) ([]*entity.Strategy, error)

	// ListActive retrieves all active strategies.
	ListActive(ctx context.Context) ([]*entity.Strategy, error)

	// Delete removes a strategy.
	// Only allowed for stopped strategies with no open positions.
	Delete(ctx context.Context, id string) error
}

// BalanceRepository defines persistence operations for balance snapshots.
type BalanceRepository interface {
	// Save persists a balance snapshot.
	Save(ctx context.Context, balance *entity.AccountBalance) error

	// GetLatest retrieves the most recent balance snapshot.
	GetLatest(ctx context.Context) (*entity.AccountBalance, error)

	// GetHistory retrieves balance snapshots within a time range.
	GetHistory(ctx context.Context, from, to time.Time) ([]*entity.AccountBalance, error)
}

// SymbolRepository defines persistence operations for symbol metadata.
type SymbolRepository interface {
	// Upsert creates or updates symbol metadata.
	Upsert(ctx context.Context, symbol *entity.Symbol) error

	// GetByName retrieves symbol metadata by name.
	GetByName(ctx context.Context, name string) (*entity.Symbol, error)

	// List retrieves all symbols.
	List(ctx context.Context) ([]*entity.Symbol, error)

	// ListByStatus retrieves symbols filtered by status.
	ListByStatus(ctx context.Context, status entity.SymbolStatus) ([]*entity.Symbol, error)
}

// RateLimitRepository defines persistence operations for rate limiter state.
type RateLimitRepository interface {
	// Save persists rate limiter bucket state.
	// Must not write more frequently than 1 second per bucket.
	Save(ctx context.Context, bucketID string, tokens int, lastRefillUTC time.Time) error

	// Load retrieves rate limiter bucket state.
	// Returns default values if the state is older than 5 minutes.
	Load(ctx context.Context, bucketID string) (tokens int, lastRefillUTC time.Time, err error)
}

// MigrationRepository defines schema migration operations.
type MigrationRepository interface {
	// GetVersion retrieves the current schema version.
	GetVersion(ctx context.Context) (version int, err error)

	// RecordMigration records a successful migration.
	RecordMigration(ctx context.Context, version int, hash string, agentVersion string) error

	// ListMigrations retrieves all applied migrations.
	ListMigrations(ctx context.Context) ([]Migration, error)
}

// Migration represents a schema migration record.
type Migration struct {
	Version      int
	Hash         string
	AgentVersion string
	AppliedAtUTC time.Time
}

// TransactionFunc is a function executed within a database transaction.
type TransactionFunc func(ctx context.Context) error

// DB defines the database interface with transaction support.
type DB interface {
	// WithTransaction executes a function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	WithTransaction(ctx context.Context, fn TransactionFunc) error

	// Close closes the database connection.
	Close() error

	// Ping verifies database connectivity.
	Ping(ctx context.Context) error
}
