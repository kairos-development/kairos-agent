package sqlite

const (
	// SchemaMigrations stores applied migration records.
	schemaMigrations = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			hash TEXT NOT NULL,
			agent_version TEXT NOT NULL,
			applied_at_utc TEXT NOT NULL
		);
	`

	// Orders table stores all order records in the critical stream.
	ordersTable = `
		CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			client_order_id TEXT NOT NULL UNIQUE,
			exchange_order_id TEXT,
			strategy_id TEXT NOT NULL,
			symbol TEXT NOT NULL,
			side TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			time_in_force TEXT NOT NULL,
			quantity TEXT NOT NULL,
			price TEXT NOT NULL,
			filled_qty TEXT NOT NULL DEFAULT '0',
			remaining_qty TEXT NOT NULL,
			avg_fill_price TEXT NOT NULL DEFAULT '0',
			created_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			submitted_at_utc TEXT,
			filled_at_utc TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_orders_strategy ON orders(strategy_id);
		CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
		CREATE INDEX IF NOT EXISTS idx_orders_exchange_id ON orders(exchange_order_id);
		CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at_utc);
	`

	// Positions table stores position records.
	positionsTable = `
		CREATE TABLE IF NOT EXISTS positions (
			id TEXT PRIMARY KEY,
			strategy_id TEXT NOT NULL,
			symbol TEXT NOT NULL,
			side TEXT NOT NULL,
			quantity TEXT NOT NULL,
			entry_price TEXT NOT NULL,
			current_price TEXT NOT NULL,
			unrealized_pnl TEXT NOT NULL DEFAULT '0',
			realized_pnl TEXT NOT NULL DEFAULT '0',
			opened_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			closed_at_utc TEXT,
			UNIQUE(strategy_id, symbol)
		);

		CREATE INDEX IF NOT EXISTS idx_positions_strategy ON positions(strategy_id);
		CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol);
		CREATE INDEX IF NOT EXISTS idx_positions_side ON positions(side);
	`

	// Strategies table stores strategy configurations.
	strategiesTable = `
		CREATE TABLE IF NOT EXISTS strategies (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			symbol TEXT NOT NULL,
			max_position_size TEXT NOT NULL,
			max_daily_loss TEXT NOT NULL,
			current_pnl TEXT NOT NULL DEFAULT '0',
			daily_pnl TEXT NOT NULL DEFAULT '0',
			total_trades INTEGER NOT NULL DEFAULT 0,
			winning_trades INTEGER NOT NULL DEFAULT 0,
			losing_trades INTEGER NOT NULL DEFAULT 0,
			created_at_utc TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL,
			started_at_utc TEXT,
			stopped_at_utc TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_strategies_status ON strategies(status);
		CREATE INDEX IF NOT EXISTS idx_strategies_type ON strategies(type);
	`

	// Balances table stores balance snapshots.
	balancesTable = `
		CREATE TABLE IF NOT EXISTS balances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			asset TEXT NOT NULL,
			total TEXT NOT NULL,
			available TEXT NOT NULL,
			locked TEXT NOT NULL,
			updated_at_utc TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_balances_asset ON balances(asset);
		CREATE INDEX IF NOT EXISTS idx_balances_updated ON balances(updated_at_utc);
	`

	// Symbols table stores symbol metadata and trading constraints.
	symbolsTable = `
		CREATE TABLE IF NOT EXISTS symbols (
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
		);

		CREATE INDEX IF NOT EXISTS idx_symbols_status ON symbols(status);
	`

	// RateState table stores rate limiter bucket state.
	rateStateTable = `
		CREATE TABLE IF NOT EXISTS rate_state (
			bucket_id TEXT PRIMARY KEY,
			tokens INTEGER NOT NULL,
			last_refill_utc TEXT NOT NULL
		);
	`
)

// Migration represents a schema migration with version and hash.
type Migration struct {
	Version int
	Hash    string
	SQL     string
}

// Migrations contains all schema migrations in order.
// Each migration must have a unique version number and deterministic hash.
var Migrations = []Migration{
	{
		Version: 1,
		Hash:    "1b13902635519aa99e1f9093f18453e51002f8237f7573eeb3443a4cdf4e1b6c", // SHA-256 of migration 1
		SQL:     schemaMigrations + ordersTable + positionsTable + strategiesTable + balancesTable + symbolsTable + rateStateTable,
	},
}
