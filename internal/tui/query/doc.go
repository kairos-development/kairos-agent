// Package query provides read-side snapshots for TUI views.
//
// QueryService returns viewmodel data by reading from domain
// repositories and runtime state. It must not perform mutations or
// call external services.
//
// This package defines the repository interfaces it reads from.
// It must not import actions, components, or controller packages.
package query
