package usecase

import (
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

// TestWorkflowOutputsUseCase_Execute_Success tests the success scenario.
func TestWorkflowOutputsUseCase_Execute_Success(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowOutputs(mockCromwell)

	input := &WorkflowOutputsInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedOutputs := entities.OutputsResponse{
		ID: "test-workflow-id",
		Outputs: map[string]interface{}{
			"task.outputKey": "some-value",
		},
	}

	// We mock CromwellClient.Outputs to return our expectedOutputs, nil error
	mockCromwell.
		On("Outputs", input.WorkflowID).
		Return(expectedOutputs, nil)

	// WHEN
	result, err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err, "Error should be nil on success")
	assert.NotNil(t, result, "Result should not be nil on success")
	assert.Equal(t, input.WorkflowID, result.WorkflowID, "WorkflowID should match input")
	assert.Equal(t, expectedOutputs, result.Outputs, "Outputs should match the expected response")

	// Ensures the mock was called as expected
	mockCromwell.AssertExpectations(t)
}

// TestWorkflowOutputsUseCase_Execute_Error tests when an error is returned by Cromwell.
func TestWorkflowOutputsUseCase_Execute_Error(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowOutputs(mockCromwell)

	input := &WorkflowOutputsInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedError := errors.New("failed to retrieve outputs")

	// CromwellClient.Outputs returns an error
	mockCromwell.
		On("Outputs", input.WorkflowID).
		Return(entities.OutputsResponse{}, expectedError)

	// WHEN
	result, err := useCase.Execute(input)

	// THEN
	assert.Error(t, err, "We expect an error")
	assert.Nil(t, result, "Result should be nil when there's an error")
	assert.Equal(t, expectedError, err, "Error should match the expected error")

	mockCromwell.AssertExpectations(t)
}
