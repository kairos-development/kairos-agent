// Package cli provides the Cobra CLI entrypoint and subcommands.
//
// CLI commands are service-only and scriptable: init, check, headless,
// backup, restore, export, emergency-stop. The CLI must not become a
// full operator UI; strategy creation belongs in TUI.
//
// This package wires controllers to the agent service layer via
// the bootstrap container.
package cli
