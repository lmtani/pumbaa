package workflow

// HealthStatus represents the health status of the Cromwell server.
type HealthStatus struct {
	OK               bool     // All subsystems are healthy
	Degraded         bool     // Some subsystems are unhealthy
	UnhealthySystems []string // List of unhealthy subsystem names
}
