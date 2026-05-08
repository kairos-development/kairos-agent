// Package engine is the core trading engine that orchestrates all
// components: state machine, event bus, NTP synchronization, circuit
// breakers, and graceful shutdown.
//
// The engine owns the runtime lifecycle and coordinates service-level
// components. It must not import controllers, TUI, or CLI code.
package engine
