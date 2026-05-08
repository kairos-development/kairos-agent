// Package backtest provides a deterministic backtesting engine with
// a fake clock and historical data replay.
//
// The engine replays historical market data through a strategy and
// records simulated trades. Same seed and market data produce identical
// results, enabling reproducible strategy evaluation.
//
// This package must not import exchange connectors or UI code.
package backtest
