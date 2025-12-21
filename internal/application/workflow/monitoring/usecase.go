// Package monitoring provides the use case for analyzing resource monitoring logs.
package monitoring

import (
	"context"

	"github.com/lmtani/pumbaa/internal/domain/workflow/monitoring"
)

// ResourceAnalysisResult contains the result of resource usage analysis.
type ResourceAnalysisResult struct {
	Report *monitoring.EfficiencyReport
}

// Usecase defines the operations for analyzing resource monitoring logs.
type Usecase interface {
	// AnalyzeResourceUsage reads a monitoring log and returns an efficiency report.
	AnalyzeResourceUsage(ctx context.Context, path string) (*ResourceAnalysisResult, error)
}

// usecaseImpl is the concrete implementation of the monitoring Usecase.
type usecaseImpl struct {
	fileProvider monitoring.FileProvider
}

// NewUsecase creates a new monitoring Usecase instance.
func NewUsecase(fp monitoring.FileProvider) Usecase {
	return &usecaseImpl{fileProvider: fp}
}

// AnalyzeResourceUsage orchestrates the reading, parsing and analysis of monitoring logs.
func (u *usecaseImpl) AnalyzeResourceUsage(ctx context.Context, path string) (*ResourceAnalysisResult, error) {
	// Read the file content using the injected file provider
	content, err := u.fileProvider.Read(ctx, path)
	if err != nil {
		return nil, err
	}

	// Parse the TSV content into metrics
	metrics, err := monitoring.ParseFromTSV(content)
	if err != nil {
		return nil, err
	}

	// Analyze the metrics and generate efficiency report
	report := monitoring.Analyze(metrics)

	return &ResourceAnalysisResult{
		Report: report,
	}, nil
}
