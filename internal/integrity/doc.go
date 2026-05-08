// Package integrity verifies runtime self-integrity by computing a
// SHA-256 hash of the binary excluding the attestation block.
//
// A tampered binary enters degraded mode: cloud updates are blocked,
// but local risk management and position closing remain available.
//
// This package must not import domain or service packages.
package integrity
