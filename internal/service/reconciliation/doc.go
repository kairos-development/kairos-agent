// Package reconciliation reconciles local order and position state with
// exchange state. It runs on startup, reconnect, WebSocket gaps, and
// operator command.
//
// The reconciler detects zombie orders, state mismatches, and partial
// fills. It pauses new entries when consistency is unknown and produces
// a reconciliation report before resuming.
//
// This package defines the ExchangeConnector interface it consumes and
// must not import controller, TUI, or storage implementation packages.
package reconciliation
