// Package strategy provides strategy execution, building, and
// WASM sandboxing for user-authored trading strategies.
//
// Strategies emit signals/intents; the risk engine converts
// approved intents into orders. Strategies never access the
// filesystem, network, or wall-clock time directly.
package strategy
