// Package app owns the bootstrapped agent subsystems: config,
// storage, vault, audit, journal, and runtime manager.
//
// Application holds the low-level infrastructure that services
// and controllers depend on. Bootstrap creates the Application
// from a state directory and vault password.
package app
