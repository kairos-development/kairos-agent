// Package sqlite contains SQLite-specific implementations of
// repository interfaces: orders, positions, balance snapshots,
// and strategy persistence.
//
// All queries use parameterized statements. Timestamps are stored
// and retrieved in UTC. Monetary values use decimal.Decimal via
// string serialization to avoid precision loss.
package sqlite
