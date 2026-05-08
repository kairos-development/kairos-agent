// Package actions provides write-side commands for the TUI.
//
// ActionService executes commands such as starting paper/live trading,
// scanning markets, queueing backtests, updating configuration, and
// exporting data. All mutations are audited.
//
// This package defines the service interfaces it consumes.
// It must not import query, components, or controller packages.
package actions
