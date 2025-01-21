package usecase

import (
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

// TestWorkflowSubmitUseCase_Execute_Success tests a successful submit scenario.
func TestWorkflowSubmitUseCase_Execute_Success(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowSubmit(mockCromwell)

	input := &WorkflowSubmitInputDTO{
		Wdl:          "workflow.wdl",
		Inputs:       "{}",
		Dependencies: "deps.zip",
		Options:      "{}",
	}

	expectedSubmitResponse := entities.SubmitResponse{
		ID:     "fake-workflow-id",
		Status: "Submitted",
	}

	// The mock should return our expectedSubmitResponse when Submit is called
	mockCromwell.
		On("Submit", input.Wdl, input.Inputs, input.Dependencies, input.Options).
		Return(expectedSubmitResponse, nil)

	// WHEN
	output, err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err, "Should not return an error on success")
	assert.NotNil(t, output, "Output should not be nil on success")
	assert.Equal(t, expectedSubmitResponse.ID, output.WorkflowID, "WorkflowID should match the mock response")
	assert.Equal(t, expectedSubmitResponse.Status, output.Status, "Status should match the mock response")

	// Ensure all expectations on the mock were met
	mockCromwell.AssertExpectations(t)
}

// TestWorkflowSubmitUseCase_Execute_Error tests when an error is returned by Cromwell Submit.
func TestWorkflowSubmitUseCase_Execute_Error(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowSubmit(mockCromwell)

	input := &WorkflowSubmitInputDTO{
		Wdl:          "broken-workflow.wdl",
		Inputs:       "{}",
		Dependencies: "",
		Options:      "{}",
	}

	expectedErr := errors.New("failed to submit workflow")

	// The mock should return an error when Submit is called
	mockCromwell.
		On("Submit", input.Wdl, input.Inputs, input.Dependencies, input.Options).
		Return(entities.SubmitResponse{}, expectedErr)

	// WHEN
	output, err := useCase.Execute(input)

	// THEN
	assert.Error(t, err, "We expect an error")
	assert.Nil(t, output, "Output should be nil when there's an error")
	assert.Equal(t, expectedErr, err, "Error should match the expected error")

	mockCromwell.AssertExpectations(t)
}
