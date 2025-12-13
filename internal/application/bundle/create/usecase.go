// Package create contains the use case for creating WDL bundles.
package create

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/domain/bundle"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// UseCase handles WDL bundle creation.
type UseCase struct{}

// New creates a new bundle creation use case.
func New() *UseCase {
	return &UseCase{}
}

// Input represents the input for the bundle creation use case.
type Input struct {
	MainWorkflowPath string
	OutputPath       string
}

// Output represents the output of the bundle creation use case.
type Output struct {
	MainWDLPath         string
	DependenciesZipPath string
	Dependencies        []string
	TotalFiles          int
}

// Execute creates a WDL bundle with all dependencies.
// It produces two files:
// 1. A main WDL file with imports rewritten to reference flattened paths
// 2. A ZIP file containing all dependencies (only if there are dependencies)
func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	// Validate input
	if input.MainWorkflowPath == "" {
		return nil, bundle.ErrMainWorkflowNotFound
	}

	if input.OutputPath == "" {
		return nil, fmt.Errorf("output path is required")
	}

	// Use the wdl package to create the bundle
	result, err := wdl.CreateBundle(input.MainWorkflowPath, input.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", bundle.ErrBundleCreationFailed, err)
	}

	// Prepare output
	return &Output{
		MainWDLPath:         result.MainWDLPath,
		DependenciesZipPath: result.DependenciesZipPath,
		Dependencies:        result.Dependencies,
		TotalFiles:          result.TotalFiles,
	}, nil
}
