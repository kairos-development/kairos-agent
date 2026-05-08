// Package tui is the terminal operator interface for Kairos.
//
// It separates the presentation into query (read-side snapshots),
// actions (write-side commands), viewmodel (presentation DTOs),
// components (pure rendering widgets), commands (command parsing),
// and styles (visual tokens).
//
// TUI must NOT directly access engine, repositories, or connector.
// Views load data only through QueryService. Commands call only
// ActionService. Components render only viewmodel data.
package tui
