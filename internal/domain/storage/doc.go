// Package storage defines repository interfaces for persistence.
//
// Repositories declare the persistence contract for orders, positions,
// strategies, and balance snapshots. Implementations live in
// internal/providers/storage/sqlite and must enforce UTC timestamps,
// append-only critical stream semantics, and full synchronous writes.
//
// This package must not import providers or controllers.
package storage
