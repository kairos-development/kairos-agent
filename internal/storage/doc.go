// Package storage owns persistent storage connections and migrations.
//
// It manages the critical SQLite stream (synchronous=FULL, explicit
// fsync) and analytics stream (synchronous=NORMAL, bounded queue).
// All migrations are versioned, hashed, and create a backup before
// applying.
//
// This package must not import domain entity definitions as runtime
// dependencies; it operates on raw storage models.
package storage
