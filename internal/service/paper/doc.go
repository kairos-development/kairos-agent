// Package paper provides a paper trading simulator that mimics exchange
// fill behavior without real orders.
//
// The paper exchange maintains virtual balances and simulates fills
// based on market prices. All monetary calculations use decimal.Decimal.
//
// This package must not import exchange connectors or UI code.
package paper
