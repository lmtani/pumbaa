// Package ports defines the interfaces for external dependencies (repositories, services).
// This file defines the interface for application update checks.
package ports

// VersionInfo describes the result of an update check.
type VersionInfo struct {
	Current         string
	Latest          string
	UpdateAvailable bool
	ReleaseURL      string
}

// UpdateChecker checks for newer releases of the application.
type UpdateChecker interface {
	// Check starts an async check for the latest version. The channel
	// receives at most one result and is closed afterwards; a nil result
	// means the check was skipped or failed.
	Check(currentVersion string) <-chan *VersionInfo
}
