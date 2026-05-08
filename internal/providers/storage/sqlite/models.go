package sqlite

import "time"

// AppliedMigration represents a schema migration record from the database.
type AppliedMigration struct {
	Version      int
	Hash         string
	AgentVersion string
	AppliedAtUTC time.Time
}
