// Package backup creates encrypted deterministic state archives using
// age encryption (scrypt passphrase).
//
// Backups are created automatically before migrations and updates, and
// can be triggered manually via CLI. The archive includes config.yaml,
// trades.sqlite, vault.db, audit.log, and journal.log with deterministic
// timestamp zeroing for reproducibility.
//
// This package must not import domain or UI code.
package backup
