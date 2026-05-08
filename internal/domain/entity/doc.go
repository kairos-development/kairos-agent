// Package entity contains pure domain models for the Kairos trading agent.
//
// All monetary calculations use github.com/shopspring/decimal.
// All timestamps are UTC. Float64 is forbidden in this package and all
// packages that depend on it.
//
// This package must not import providers, controllers, service packages,
// or any infrastructure code. It defines the canonical shape of Orders,
// Positions, Balances, Strategies, and Symbols used throughout the agent.
package entity
