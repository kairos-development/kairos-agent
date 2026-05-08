// Package risk contains non-disableable pre-trade risk checks.
//
// The risk engine is mandatory per architecture baseline and cannot be
// disabled by configuration. All monetary calculations use decimal.Decimal.
//
// This package is pure domain logic: it must not import providers,
// controllers, exchange connectors, storage implementations, or TUI code.
package risk
