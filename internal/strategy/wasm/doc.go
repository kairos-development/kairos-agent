// Package wasm provides the WebAssembly runtime for executing
// user-authored trading strategies in a sandboxed environment.
//
// Strategies run with deterministic context only: no wall-clock,
// no filesystem, no network, no direct exchange access.
package wasm
