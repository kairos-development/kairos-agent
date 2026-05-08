// Package connector defines the exchange connectivity interface and
// domain-level data types for order updates, position updates, balance
// updates, and ticker data.
//
// Implementations live in internal/providers/bybit and must convert
// exchange-specific data models into these domain types.
//
// This package must not import service, providers, or controllers.
package connector
