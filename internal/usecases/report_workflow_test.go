package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

// MockWorkflowProvider is already defined in query_workflows_test.go

func TestReportWorkflow_Execute_Success(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &ReportWorkflow{
		WorkflowProvider: mockProvider,
	}

	now := time.Now()
	mockWorkflow := entities.Workflow{
		ID:     "workflow-123",
		Name:   "Test Workflow",
		Start:  now,
		End:    now.Add(time.Hour * 2),
		Status: "Succeeded",
		Calls:  map[string][]entities.Step{},
	}

	mockProvider.On("Get", "workflow-123").Return(mockWorkflow, nil)

	// Act
	result, err := usecase.Execute("workflow-123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "workflow-123", result.WorkflowID)
	assert.Equal(t, "Test Workflow", result.Name)
	assert.Equal(t, "Succeeded", result.Status)
	assert.Equal(t, now.Format(time.RFC3339), result.Start)
	assert.Equal(t, now.Add(time.Hour*2).Format(time.RFC3339), result.End)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}

func TestReportWorkflow_Execute_ProviderError(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &ReportWorkflow{
		WorkflowProvider: mockProvider,
	}

	expectedError := errors.New("provider error")
	mockProvider.On("Get", "workflow-123").Return(entities.Workflow{}, expectedError)

	// Act
	result, err := usecase.Execute("workflow-123")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}

func TestReportWorkflow_Execute_WorkflowNotFound(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &ReportWorkflow{
		WorkflowProvider: mockProvider,
	}

	// Return an empty workflow with no error - simulating a case where the provider
	// doesn't return an error but the workflow is not found
	mockProvider.On("Get", "workflow-123").Return(entities.Workflow{}, nil)

	// Act
	result, err := usecase.Execute("workflow-123")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, "workflow not found", err.Error())
	assert.Nil(t, result)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}

func TestReportWorkflow_Execute_DifferentWorkflowIDs(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &ReportWorkflow{
		WorkflowProvider: mockProvider,
	}

	now := time.Now()
	mockWorkflow := entities.Workflow{
		ID:     "workflow-456",
		Name:   "Another Workflow",
		Start:  now,
		End:    now.Add(time.Hour),
		Status: "Running",
		Calls:  map[string][]entities.Step{},
	}

	mockProvider.On("Get", "workflow-456").Return(mockWorkflow, nil)

	// Act
	result, err := usecase.Execute("workflow-456")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "workflow-456", result.WorkflowID)
	assert.Equal(t, "Another Workflow", result.Name)
	assert.Equal(t, "Running", result.Status)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}
