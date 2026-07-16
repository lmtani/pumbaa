// Package ports defines the interfaces for external dependencies (repositories, services).
// This file defines the interface for rendering the resource visualization report.
package ports

// ResourceReportData carries the pre-serialized payloads for the resource report.
type ResourceReportData struct {
	DataJSON            []byte // Raw task data as JSON
	WorkflowsJSON       []byte // List of workflow IDs as JSON
	RecommendationsJSON []byte // Task recommendations as JSON
	LLMModelInfo        string // LLM provider and model used for recommendations
}

// ResourceReportRenderer renders the resource visualization report (e.g. HTML).
type ResourceReportRenderer interface {
	Render(data ResourceReportData) (string, error)
}
