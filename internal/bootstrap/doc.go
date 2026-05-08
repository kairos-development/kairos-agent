// Package bootstrap wires application dependencies into a layered
// container: Application (low-level subsystems) and AgentService
// (use-case orchestration).
//
// The Factory is the single composition root. All concrete dependencies
// are constructed here; services and controllers receive interfaces.
//
// This package must not import controller, TUI, or CLI packages.
package bootstrap
