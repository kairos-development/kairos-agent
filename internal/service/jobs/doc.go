// Package jobs provides a job manager with priority classes for
// orchestrating background work.
//
// Priority classes: critical (order routing, risk, reconciliation),
// live (market data, strategy ticks), background (metadata refresh,
// backups), analytics (metrics aggregation, export).
//
// Critical jobs cannot be dropped. Analytics jobs may be dropped
// oldest-first under pressure. The job manager tracks CPU/RAM/I/O
// budget and degrades analytics before trading safety.
//
// This package must not import domain or controller packages.
package jobs
