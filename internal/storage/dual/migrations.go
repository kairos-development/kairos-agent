package dual

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func RunMigrations(ctx context.Context, db *sqlx.DB) error {
	migrations := []struct {
		name string
		fn   func(*sqlx.DB) error
	}{
		{"001_create_analytics_events", migrateCreateAnalyticsEvents},
		{"002_create_critical_tables", migrateCreateCriticalTables},
	}

	for _, m := range migrations {
		if err := m.fn(db); err != nil {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
	}
	return nil
}

func migrateCreateAnalyticsEvents(db *sqlx.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS analytics_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		data TEXT,
		timestamp DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_analytics_events_type ON analytics_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_analytics_events_timestamp ON analytics_events(timestamp);
	`
	_, err := db.Exec(query)
	return err
}

func migrateCreateCriticalTables(db *sqlx.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_order_id TEXT UNIQUE NOT NULL,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL,
		quantity TEXT NOT NULL,
		price TEXT,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

	CREATE TABLE IF NOT EXISTS balances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		asset TEXT NOT NULL,
		free TEXT NOT NULL,
		locked TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_balances_asset ON balances(asset);

	CREATE TABLE IF NOT EXISTS risk_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_risk_events_type ON risk_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_risk_events_created ON risk_events(created_at);
	`
	_, err := db.Exec(query)
	return err
}

func ExecInTransaction(ctx context.Context, db *sqlx.DB, fn func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			err = fmt.Errorf("transaction panic recovered: %v", p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
