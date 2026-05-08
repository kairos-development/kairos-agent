// Package journal manages the append-only journal log.
//
// The journal records risk-approved order intents before exchange
// submission, enabling idempotent replay and crash recovery.
// ClientOrderID is the deduplication key during replay.
//
// This package must not import exchange connectors or UI code.
package journal
