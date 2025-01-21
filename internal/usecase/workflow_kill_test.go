package usecase

import (
	"errors"
	"testing"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockCromwellServer is a mock implementation of the entities.CromwellServer interface.
type mockCromwellServer struct {
	mock.Mock
}

func (m *mockCromwellServer) Kill(workflowID string) (entities.SubmitResponse, error) {
	args := m.Called(workflowID)
	killResult, _ := args.Get(0).(entities.SubmitResponse)
	return killResult, args.Error(1)
}

func (m *mockCromwellServer) Status(workflowID string) (entities.SubmitResponse, error) {
	args := m.Called(workflowID)
	statusResult, _ := args.Get(0).(entities.SubmitResponse)
	return statusResult, args.Error(1)
}

func (m *mockCromwellServer) Outputs(workflowID string) (entities.OutputsResponse, error) {
	args := m.Called(workflowID)
	outputsResult, _ := args.Get(0).(entities.OutputsResponse)
	return outputsResult, args.Error(1)
}

func (m *mockCromwellServer) Query(params *entities.ParamsQueryGet) (entities.QueryResponse, error) {
	args := m.Called(params)
	queryResult, _ := args.Get(0).(entities.QueryResponse)
	return queryResult, args.Error(1)
}

func (m *mockCromwellServer) Metadata(workflowID string, params *entities.ParamsMetadataGet) (entities.MetadataResponse, error) {
	args := m.Called(workflowID, params)
	metadataResult, _ := args.Get(0).(entities.MetadataResponse)
	return metadataResult, args.Error(1)
}

func (m *mockCromwellServer) Submit(wdl, inputs, dependencies, options string) (entities.SubmitResponse, error) {
	args := m.Called(wdl, inputs, dependencies, options)
	submitResult, _ := args.Get(0).(entities.SubmitResponse)
	return submitResult, args.Error(1)
}

func TestWorkflowKillUseCase_Execute_Success(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	usecase := NewWorkflowKill(mockCromwell)

	input := &WorkflowKillInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedResult := entities.SubmitResponse{
		Status: "success",
	}

	// We tell our mock that we expect CromwellServer.Kill to be called with "test-workflow-id"
	// and we want it to return our expectedResult, nil for error
	mockCromwell.
		On("Kill", input.WorkflowID).
		Return(expectedResult, nil)

	// WHEN
	output, err := usecase.Execute(input)

	// THEN
	assert.NoError(t, err, "Error should be nil")
	assert.NotNil(t, output, "Output should not be nil")
	assert.Equal(t, expectedResult.Status, output.Status, "Status should match the mock result")

	// Ensures all expectations on the mock were met
	mockCromwell.AssertExpectations(t)
}

func TestWorkflowKillUseCase_Execute_Error(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	usecase := NewWorkflowKill(mockCromwell)

	input := &WorkflowKillInputDTO{
		WorkflowID: "test-workflow-id",
	}

	// We simulate an error returned by the Cromwell server
	expectedError := errors.New("failed to kill workflow")

	mockCromwell.
		On("Kill", input.WorkflowID).
		Return(entities.SubmitResponse{}, expectedError)

	// WHEN
	output, err := usecase.Execute(input)

	// THEN
	assert.Error(t, err, "We expect an error")
	assert.Nil(t, output, "Output should be nil when there's an error")
	assert.Equal(t, expectedError, err, "Error should match the mock error")

	mockCromwell.AssertExpectations(t)
}
