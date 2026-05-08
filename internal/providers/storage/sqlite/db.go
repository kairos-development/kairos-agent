package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var (
	// ErrNotFound reports that the requested entity was not found.
	ErrNotFound = errors.New("entity not found")

	// ErrDuplicateKey reports a unique constraint violation.
	ErrDuplicateKey = errors.New("duplicate key violation")
)

// DB wraps a SQLite database connection with transaction support.
type DB struct {
	conn *sql.DB
	path string
}

// Open opens a SQLite database at the specified path.
// The database is created if it does not exist.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(1) // SQLite supports only one writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	return &DB{
		conn: conn,
		path: path,
	}, nil
}

// OpenCritical opens a SQLite database with PRAGMA synchronous=FULL for critical stream.
// This ensures fsync after every transaction for durability.
func OpenCritical(path string) (*DB, error) {
	db, err := Open(path)
	if err != nil {
		return nil, err
	}

	// Enable full synchronous mode for critical stream
	if _, err := db.conn.Exec("PRAGMA synchronous=FULL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set synchronous=FULL: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set journal_mode=WAL: %w", err)
	}

	return db, nil
}

// OpenAnalytics opens a SQLite database with PRAGMA synchronous=NORMAL for analytics stream.
// This provides better performance for non-critical data.
func OpenAnalytics(path string) (*DB, error) {
	db, err := Open(path)
	if err != nil {
		return nil, err
	}

	// Use normal synchronous mode for analytics
	if _, err := db.conn.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set synchronous=NORMAL: %w", err)
	}

	// Enable WAL mode
	if _, err := db.conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set journal_mode=WAL: %w", err)
	}

	return db, nil
}

// WithTransaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *DB) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Store transaction in context for repository access
	ctx = context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback after error %v: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Ping verifies database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

// Conn returns the underlying sql.DB connection.
// This is used by repositories to execute queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// txKey is a context key for storing the current transaction.
type txKey struct{}

// getTx retrieves the transaction from context, or returns the connection if no transaction is active.
func getTx(ctx context.Context, conn *sql.DB) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return conn
}

// utcNow returns the current time in UTC as a string for SQLite storage.
func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// parseUTC parses a UTC timestamp string from SQLite.
func parseUTC(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}

// isConstraintError returns true if the error is a unique constraint violation.
func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite constraint error messages contain "constraint failed"
	errMsg := err.Error()
	return errors.Is(err, sql.ErrNoRows) == false &&
		(errors.Is(err, errors.New("constraint failed")) ||
			len(errMsg) > 0 && (len(errMsg) >= 18 && errMsg[:18] == "constraint failed:" ||
				len(errMsg) >= 16 && errMsg[:16] == "UNIQUE constraint" ||
				len(errMsg) >= 19 && errMsg[:19] == "PRIMARY KEY constraint"))
}
