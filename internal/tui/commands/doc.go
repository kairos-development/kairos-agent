// Package commands parses and routes slash-command text from the
// TUI command input to ActionService calls.
//
// The handler validates command names and arguments, then delegates
// to the appropriate action. It must not render UI or import
// component packages.
package commands
