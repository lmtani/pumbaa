// Package ports defines the interfaces for external dependencies (repositories, services).
// This file defines the interface for resource optimization recommendation generation.
package ports

import (
	"context"
)

// TaskAnalysisData contains all the data needed to analyze a task's resource usage.
// This is the input for the RecommendationGenerator.
type TaskAnalysisData struct {
	TaskName        string             `json:"taskName"`        // WDL task name (not alias)
	SampleCount     int                `json:"sampleCount"`     // Number of samples analyzed
	InputSizes      map[string][]int64 `json:"inputSizes"`      // Input name -> sizes per sample (bytes)
	DiskPeaksGB     []float64          `json:"diskPeaksGB"`     // Disk usage per sample (GB)
	MemoryPeaksMB   []float64          `json:"memoryPeaksMB"`   // Memory usage per sample (MB)
	CPUMeans        []float64          `json:"cpuMeans"`        // CPU usage per sample (%)
	DurationSeconds []float64          `json:"durationSeconds"` // Duration per sample (seconds)
	// Resource requests (from runtime attributes)
	CPURequest   string  `json:"cpuRequest"`   // Configured CPU
	MemoryReqGB  float64 `json:"memoryReqGB"`  // Configured memory (GB)
	DiskReqGB    float64 `json:"diskReqGB"`    // Configured disk (GB)
	ResourceCost float64 `json:"resourceCost"` // Computed resource cost for prioritization
}

// RecommendationSeverity indicates the urgency of a recommendation.
type RecommendationSeverity string

const (
	SeverityGood     RecommendationSeverity = "good"     // Green - resource is well-utilized
	SeverityWarning  RecommendationSeverity = "warning"  // Yellow - needs attention
	SeverityCritical RecommendationSeverity = "critical" // Red - critical issue
)

// RecommendationItem represents a single recommendation with severity.
type RecommendationItem struct {
	Message  string                 `json:"message"`
	Severity RecommendationSeverity `json:"severity"` // good, warning, critical
}

// TaskRecommendation contains optimization recommendations for a task.
type TaskRecommendation struct {
	TaskName        string                 `json:"taskName"`
	SampleCount     int                    `json:"sampleCount"`
	OverallStatus   RecommendationSeverity `json:"overallStatus"` // LLM-determined: good, warning, critical
	ResourceCost    float64                `json:"resourceCost"`  // Total dimensionless cost for prioritization
	CPUCost         float64                `json:"cpuCost"`       // CPU contribution
	MemoryCost      float64                `json:"memoryCost"`    // Memory contribution
	DiskCost        float64                `json:"diskCost"`      // Disk contribution
	DiskFormula     string                 `json:"diskFormula,omitempty"`
	DiskR2          float64                `json:"diskR2,omitempty"`
	MemoryFormula   string                 `json:"memoryFormula,omitempty"`
	MemoryR2        float64                `json:"memoryR2,omitempty"`
	Recommendations []RecommendationItem   `json:"recommendations"` // Changed from []string
}

// RecommendationGenerator generates resource optimization recommendations for tasks.
// Implementations may use statistical analysis, LLM, or other methods.
type RecommendationGenerator interface {
	// GenerateRecommendations analyzes task data and returns optimization suggestions.
	// The implementation may use tools to look up WDL definitions for context.
	GenerateRecommendations(ctx context.Context, tasks []TaskAnalysisData) ([]TaskRecommendation, error)

	// IsAvailable returns true if the generator is properly configured and ready to use.
	// If false, the caller should proceed without recommendations.
	IsAvailable() bool
}
