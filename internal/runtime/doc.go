// Package runtime owns mutable runtime state: run mode, connectivity,
// license, integrity, NTP drift, and degradation flags.
//
// It provides thread-safe access to the agent's operational status and
// enforces safety gates such as disclaimer acceptance before live trading.
//
// This package must not import domain, service, or providers.
package runtime
