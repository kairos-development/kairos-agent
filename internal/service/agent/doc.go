// Package agent provides application-layer use cases for the trading
// agent: starting and stopping trading modes, managing configuration,
// and exporting data.
//
// It orchestrates domain services, repository access, and runtime
// state transitions. This package defines the repository interfaces
// it consumes and must not import controller or UI packages.
package agent
