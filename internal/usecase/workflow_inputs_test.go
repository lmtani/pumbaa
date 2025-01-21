package usecase

import (
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowInputsUseCase_Execute_Success(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	usecase := NewWorkflowInputs(mockCromwell)

	input := &WorkflowInputsInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedMetadata := entities.MetadataResponse{
		Inputs: map[string]interface{}{
			"foo": "bar",
			"num": 123,
		},
		// Other fields are zero-valued or irrelevant here
	}

	// We expect Metadata to be called with `workflowID`, and `nil` params
	mockCromwell.
		On("Metadata", input.WorkflowID, (*entities.ParamsMetadataGet)(nil)).
		Return(expectedMetadata, nil)

	// WHEN
	output, err := usecase.Execute(input)

	// THEN
	assert.NoError(t, err, "Error should be nil on success")
	assert.NotNil(t, output, "Output should not be nil on success")
	assert.Equal(t, input.WorkflowID, output.WorkflowID, "WorkflowID should match")
	assert.Equal(t, expectedMetadata.Inputs, output.Inputs, "Inputs should match the metadata returned")

	// Ensures all expectations on the mock were met
	mockCromwell.AssertExpectations(t)
}

func TestWorkflowInputsUseCase_Execute_Error(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	usecase := NewWorkflowInputs(mockCromwell)

	input := &WorkflowInputsInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedError := errors.New("failed to retrieve metadata")

	// We expect Metadata to return an error this time
	mockCromwell.
		On("Metadata", input.WorkflowID, (*entities.ParamsMetadataGet)(nil)).
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
