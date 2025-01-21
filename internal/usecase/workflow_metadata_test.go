package usecase

import (
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowMetadataUseCase_Execute_Success(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	usecase := NewWorkflowMetadata(mockCromwell)

	input := &WorkflowMetadataInputDTO{
		WorkflowID: "test-workflow-id",
	}

	// These are the params the use case will pass to Metadata internally
	expectedParams := entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}

	expectedMetadata := entities.MetadataResponse{
		WorkflowName: "my-test-workflow",
		Status:       "Succeeded",
		// Other fields omitted for brevity
	}

	// EXPECT
	mockCromwell.
		On("Metadata", input.WorkflowID, &expectedParams).
		Return(expectedMetadata, nil)

	// WHEN
	output, err := usecase.Execute(input)

	// THEN
	assert.NoError(t, err, "Error should be nil on success")
	assert.NotNil(t, output, "Output should not be nil on success")
	assert.Equal(t, input.WorkflowID, output.WorkflowID, "WorkflowID should match input")
	assert.Equal(t, expectedMetadata, output.Metadata, "Metadata should match the expected result")

	// Ensures all expectations on the mock were met
	mockCromwell.AssertExpectations(t)
}

func TestWorkflowMetadataUseCase_Execute_Error(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	usecase := NewWorkflowMetadata(mockCromwell)

	input := &WorkflowMetadataInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedParams := entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "jes", "inputs"},
	}

	// Simulate an error
	expectedError := errors.New("failed to retrieve metadata")

	// EXPECT
	mockCromwell.
		On("Metadata", input.WorkflowID, &expectedParams).
		Return(entities.MetadataResponse{}, expectedError)

	// WHEN
	output, err := usecase.Execute(input)

	// THEN
	assert.Error(t, err, "We expect an error")
	assert.Nil(t, output, "Output should be nil when there's an error")
	assert.Equal(t, expectedError, err, "Error should match the expected error")

	// Ensures all expectations on the mock were met
	mockCromwell.AssertExpectations(t)
}
