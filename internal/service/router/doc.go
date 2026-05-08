// Package router routes validated orders to paper or live execution.
//
// It generates deterministic ClientOrderID via SHA256(strategy_id, symbol,
// side, timestamp_bucket, nonce), persists an in-flight journal entry
// before the exchange call, and supports idempotent submission.
//
// This package defines the ExchangeConnector and PaperSimulator
// interfaces it consumes. It must not import controller or UI packages.
package router
