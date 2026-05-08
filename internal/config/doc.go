// Package config manages agent configuration: loading, validation,
// atomic reload, and persistence.
//
// The config schema is versioned. Configuration changes affecting
// risk or live mode write audit events. Secrets are references to
// vault entries, not raw config values.
//
// Config load order: defaults -> file -> env -> CLI flags -> runtime
// safe overrides. This package must not import service or controllers.
package config
