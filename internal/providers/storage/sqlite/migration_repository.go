package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// MigrationRepository implements domain storage.MigrationRepository for SQLite.
type MigrationRepository struct {
	db *DB
}

// NewMigrationRepository creates a new SQLite migration repository.
func NewMigrationRepository(db *DB) *MigrationRepository {
	return &MigrationRepository{db: db}
}

// GetVersion retrieves the current schema version.
func (r *MigrationRepository) GetVersion(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`

	var version int
	err := r.db.Conn().QueryRowContext(ctx, query).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("get schema version: %w", err)
	}

	return version, nil
}

// RecordMigration records a successful migration.
func (r *MigrationRepository) RecordMigration(ctx context.Context, version int, hash string, agentVersion string) error {
	query := `
		INSERT INTO schema_migrations (version, hash, agent_version, applied_at_utc)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.Conn().ExecContext(ctx, query,
		version,
		hash,
		agentVersion,
		time.Now().UTC().Format(time.RFC3339Nano),
	)

	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return nil
}

// ListMigrations retrieves all applied migrations.
func (r *MigrationRepository) ListMigrations(ctx context.Context) ([]AppliedMigration, error) {
	query := `
		SELECT version, hash, agent_version, applied_at_utc
		FROM schema_migrations
		ORDER BY version ASC
	`

	rows, err := r.db.Conn().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	defer rows.Close()

	var migrations []AppliedMigration
	for rows.Next() {
		var (
			m         AppliedMigration
			appliedAt string
		)

		err := rows.Scan(&m.Version, &m.Hash, &m.AgentVersion, &appliedAt)
		if err != nil {
			return nil, fmt.Errorf("scan migration: %w", err)
		}

		m.AppliedAtUTC, _ = time.Parse(time.RFC3339Nano, appliedAt)
		migrations = append(migrations, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migrations: %w", err)
	}

	return migrations, nil
}

// ApplyMigrations applies all pending migrations sequentially.
// Returns an error if any migration fails, blocking runtime startup.
func ApplyMigrations(ctx context.Context, db *DB, agentVersion string) error {
	// Ensure schema_migrations table exists
	if _, err := db.Conn().ExecContext(ctx, schemaMigrations); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	repo := NewMigrationRepository(db)
	currentVersion, err := repo.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	for _, migration := range Migrations {
		if migration.Version <= currentVersion {
			continue
		}

		// Verify migration hash
		computedHash := computeMigrationHash(migration.SQL)
		if computedHash != migration.Hash {
			return fmt.Errorf("migration %d hash mismatch: expected %s, got %s",
				migration.Version, migration.Hash, computedHash)
		}

		// Apply migration
		if _, err := db.Conn().ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply migration %d: %w", migration.Version, err)
		}

		// Record migration
		if err := repo.RecordMigration(ctx, migration.Version, migration.Hash, agentVersion); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

func computeMigrationHash(sql string) string {
	hash := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(hash[:])
}
