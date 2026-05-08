// Package vault provides secure storage for API keys and other
// sensitive material using Argon2id key derivation and AES-256-GCM
// authenticated encryption.
//
// The vault is locked by default. LiveTrading requires an explicit
// Unlock with a user-provided password. Secrets are never logged
// and the password is zeroed after use where practical.
//
// This package must not import service or controllers.
package vault
