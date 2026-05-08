package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	_ "modernc.org/sqlite"
)

var errNoRateState = errors.New("rate state not found")

type migration struct {
	version int
	hash    string
	query   string
}

var tradesMigrations = []migration{{
	version: 1,
	hash:    "trades-v1-20260429",
	query: `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	hash TEXT NOT NULL,
	applied_at_utc TEXT NOT NULL,
	agent_version TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS partial_fills (
	client_order_id TEXT PRIMARY KEY,
	symbol TEXT NOT NULL,
	filled_qty TEXT NOT NULL,
	remaining_qty TEXT NOT NULL,
	avg_fill_price TEXT NOT NULL,
	updated_at_utc TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS closed_trades (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date_utc TEXT NOT NULL,
	pair TEXT NOT NULL,
	side TEXT NOT NULL,
	size TEXT NOT NULL,
	entry_price TEXT NOT NULL,
	exit_price TEXT NOT NULL,
	fee TEXT NOT NULL,
	realized_pnl TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS rate_state (
	bucket_id TEXT PRIMARY KEY CHECK(length(bucket_id) <= 128),
	tokens TEXT NOT NULL,
	capacity TEXT NOT NULL,
	refill_per_second TEXT NOT NULL,
	updated_at_utc TEXT NOT NULL
);
`,
}}

// Store owns the persistent storage connections.
type Store struct {
	tradesPath  string
	criticalDB  *sql.DB
	analyticsDB *sql.DB
	tradesFile  *os.File
	mu          sync.Mutex
}

// Open opens the trades database and applies migrations.
func Open(stateDir string, agentVersion string) (*Store, error) {
	tradesPath := filepath.Join(stateDir, "trades.sqlite")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	preExisting := fileExists(tradesPath)
	if preExisting {
		needsMigration, err := hasPendingMigrations(tradesPath, tradesMigrations)
		if err != nil {
			return nil, err
		}
		if needsMigration {
			if err := createSQLiteBackup(stateDir, tradesPath, "pre-migrate"); err != nil {
				return nil, err
			}
		}
	}
	criticalDB, err := sql.Open("sqlite", tradesPath)
	if err != nil {
		return nil, fmt.Errorf("open trades sqlite: %w", err)
	}
	analyticsDB, err := sql.Open("sqlite", tradesPath)
	if err != nil {
		criticalDB.Close()
		return nil, fmt.Errorf("open analytics sqlite: %w", err)
	}
	criticalDB.SetMaxOpenConns(1)
	if _, err := criticalDB.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("configure critical sqlite: %w", err)
	}
	if _, err := analyticsDB.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("configure analytics sqlite: %w", err)
	}
	if err := applyMigrations(criticalDB, agentVersion, tradesMigrations); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(tradesPath, os.O_RDONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open trades sqlite handle: %w", err)
	}
	return &Store{tradesPath: tradesPath, criticalDB: criticalDB, analyticsDB: analyticsDB, tradesFile: file}, nil
}

// Close closes database resources.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	var errs []string
	if s.criticalDB != nil {
		if err := s.criticalDB.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if s.analyticsDB != nil {
		if err := s.analyticsDB.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if s.tradesFile != nil {
		if err := s.tradesFile.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// WithCriticalTx runs a critical transaction and fsyncs the trades file on success.
func (s *Store) WithCriticalTx(ctx context.Context, fn func(*sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.criticalDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := s.tradesFile.Sync(); err != nil {
		return err
	}
	walFile, err := os.OpenFile(s.tradesPath+"-wal", os.O_RDONLY, 0o600)
	if err == nil {
		walFile.Sync()
		walFile.Close()
	}
	return nil
}

// SaveRateState persists a token bucket state.
func (s *Store) SaveRateState(ctx context.Context, state RateState) error {
	return s.WithCriticalTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO rate_state (bucket_id, tokens, capacity, refill_per_second, updated_at_utc)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(bucket_id) DO UPDATE SET
	tokens = excluded.tokens,
	capacity = excluded.capacity,
	refill_per_second = excluded.refill_per_second,
	updated_at_utc = excluded.updated_at_utc`,
			state.BucketID,
			state.Tokens.String(),
			state.Capacity.String(),
			state.RefillPerSecond.String(),
			state.UpdatedAtUTC.UTC().Format(time.RFC3339Nano),
		)
		return err
	})
}

// LoadRateState loads the persisted token bucket state.
func (s *Store) LoadRateState(ctx context.Context, bucketID string) (RateState, error) {
	var row RateState
	var tokens, capacity, refill, updated string
	err := s.criticalDB.QueryRowContext(ctx, `SELECT bucket_id, tokens, capacity, refill_per_second, updated_at_utc FROM rate_state WHERE bucket_id = ?`, bucketID).Scan(&row.BucketID, &tokens, &capacity, &refill, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return RateState{}, errNoRateState
	}
	if err != nil {
		return RateState{}, err
	}
	var parseErr error
	if row.Tokens, parseErr = decimal.NewFromString(tokens); parseErr != nil {
		return RateState{}, parseErr
	}
	if row.Capacity, parseErr = decimal.NewFromString(capacity); parseErr != nil {
		return RateState{}, parseErr
	}
	if row.RefillPerSecond, parseErr = decimal.NewFromString(refill); parseErr != nil {
		return RateState{}, parseErr
	}
	if row.UpdatedAtUTC, parseErr = time.Parse(time.RFC3339Nano, updated); parseErr != nil {
		return RateState{}, parseErr
	}
	return row, nil
}

// SavePartialFill stores the latest partial fill state.
func (s *Store) SavePartialFill(ctx context.Context, fill PartialFill) error {
	return s.WithCriticalTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO partial_fills (client_order_id, symbol, filled_qty, remaining_qty, avg_fill_price, updated_at_utc)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(client_order_id) DO UPDATE SET
	symbol = excluded.symbol,
	filled_qty = excluded.filled_qty,
	remaining_qty = excluded.remaining_qty,
	avg_fill_price = excluded.avg_fill_price,
	updated_at_utc = excluded.updated_at_utc`,
			fill.ClientOrderID,
			fill.Symbol,
			fill.FilledQty.String(),
			fill.RemainingQty.String(),
			fill.AvgFillPrice.String(),
			fill.UpdatedAtUTC.UTC().Format(time.RFC3339Nano),
		)
		return err
	})
}

// ListExportTrades returns closed trades for CSV export.
func (s *Store) ListExportTrades(ctx context.Context) ([]ExportTradeRecord, error) {
	rows, err := s.analyticsDB.QueryContext(ctx, `SELECT date_utc, pair, side, size, entry_price, exit_price, fee, realized_pnl FROM closed_trades ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExportTradeRecord
	for rows.Next() {
		var date, size, entry, exit, fee, pnl string
		var record ExportTradeRecord
		if err := rows.Scan(&date, &record.Pair, &record.Side, &size, &entry, &exit, &fee, &pnl); err != nil {
			return nil, err
		}
		if record.Date, err = time.Parse(time.RFC3339Nano, date); err != nil {
			return nil, err
		}
		if record.Size, err = decimal.NewFromString(size); err != nil {
			return nil, err
		}
		if record.EntryPrice, err = decimal.NewFromString(entry); err != nil {
			return nil, err
		}
		if record.ExitPrice, err = decimal.NewFromString(exit); err != nil {
			return nil, err
		}
		if record.Fee, err = decimal.NewFromString(fee); err != nil {
			return nil, err
		}
		if record.RealizedPNL, err = decimal.NewFromString(pnl); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func applyMigrations(db *sql.DB, agentVersion string, migrations []migration) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, hash TEXT NOT NULL, applied_at_utc TEXT NOT NULL, agent_version TEXT NOT NULL);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	currentVersion, err := currentVersion(db)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migration.query); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, hash, applied_at_utc, agent_version) VALUES (?, ?, ?, ?)`, migration.version, migration.hash, time.Now().UTC().Format(time.RFC3339Nano), agentVersion); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", migration.version, err)
		}
	}
	return nil
}

func currentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version)
	return version, err
}

func hasPendingMigrations(path string, migrations []migration) (bool, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return false, err
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, hash TEXT NOT NULL, applied_at_utc TEXT NOT NULL, agent_version TEXT NOT NULL);`); err != nil {
		return false, err
	}
	current, err := currentVersion(db)
	if err != nil {
		return false, err
	}
	latest := 0
	for _, migration := range migrations {
		if migration.version > latest {
			latest = migration.version
		}
	}
	return current < latest, nil
}

func createSQLiteBackup(stateDir string, source string, prefix string) error {
	if !fileExists(source) {
		return nil
	}
	backupDir := filepath.Join(stateDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(backupDir, fmt.Sprintf("%s-%s-%d.bak", prefix, filepath.Base(source), time.Now().UTC().Unix()))
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
