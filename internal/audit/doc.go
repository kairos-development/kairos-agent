// Package audit appends immutable audit records to an append-only log.
//
// Every config change, disclaimer acceptance, telemetry consent change,
// and security-relevant action writes an audit record with UTC timestamp,
// operator source, and state hash.
//
// The audit log is excluded from retention cleanup and must never be
// truncated. This package must not import domain or service packages.
package audit
