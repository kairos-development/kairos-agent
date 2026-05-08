// Package license verifies Ed25519-signed JWT license tokens offline
// using an embedded public key.
//
// It supports grace periods and RiskOnly mode: an expired or missing
// license blocks new paid/live entries but never prevents closing
// existing positions or emergency safety actions.
//
// This package must not import exchange connectors or UI code.
package license
