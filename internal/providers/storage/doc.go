// Package storage implements repository interfaces with a dual-stream
// SQLite backend.
//
// The critical stream uses synchronous=FULL with explicit fsync for
// orders, positions, and risk decisions. The analytics stream uses
// synchronous=NORMAL for ticks, candles, and metrics snapshots.
//
// This package must not import service, controller, or UI packages.
package storageprovider
