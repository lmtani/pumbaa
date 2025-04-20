package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkflowProvider is a mock implementation of the WorkflowProvider interface
type MockWorkflowProvider struct {
	mock.Mock
}

// Query mocks the Query method of WorkflowProvider
func (m *MockWorkflowProvider) Query() ([]entities.Workflow, error) {
	args := m.Called()
	return args.Get(0).([]entities.Workflow), args.Error(1)
}

// Get mocks the Get method of WorkflowProvider
func (m *MockWorkflowProvider) Get(uuid string) (entities.Workflow, error) {
	args := m.Called(uuid)
	return args.Get(0).(entities.Workflow), args.Error(1)
}

func TestQueryWorkflows_Execute_Success(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &QueryWorkflows{
		WorkflowProvider: mockProvider,
	}

	now := time.Now()
	mockWorkflows := []entities.Workflow{
		{
			ID:     "workflow-1",
			Name:   "Test Workflow 1",
			Start:  now,
			End:    now.Add(time.Hour),
			Status: "Succeeded",
			Calls:  map[string][]entities.Step{},
		},
		{
			ID:     "workflow-2",
			Name:   "Test Workflow 2",
			Start:  now,
			End:    now.Add(time.Hour * 2),
			Status: "Running",
			Calls:  map[string][]entities.Step{},
		},
	}

	mockProvider.On("Query").Return(mockWorkflows, nil)

	// Act
	result, err := usecase.Execute()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Workflows, 2)

	// Check first workflow
	assert.Equal(t, "workflow-1", result.Workflows[0].ID)
	assert.Equal(t, "Test Workflow 1", result.Workflows[0].Name)
	assert.Equal(t, "Succeeded", result.Workflows[0].Status)

	// Check second workflow
	assert.Equal(t, "workflow-2", result.Workflows[1].ID)
	assert.Equal(t, "Test Workflow 2", result.Workflows[1].Name)
	assert.Equal(t, "Running", result.Workflows[1].Status)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}

func TestQueryWorkflows_Execute_Error(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &QueryWorkflows{
		WorkflowProvider: mockProvider,
	}

	expectedError := errors.New("provider error")
	mockProvider.On("Query").Return([]entities.Workflow{}, expectedError)

	// Act
	result, err := usecase.Execute()

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}

func TestQueryWorkflows_Execute_EmptyResult(t *testing.T) {
	// Arrange
	mockProvider := new(MockWorkflowProvider)
	usecase := &QueryWorkflows{
		WorkflowProvider: mockProvider,
	}

	mockProvider.On("Query").Return([]entities.Workflow{}, nil)

	// Act
	result, err := usecase.Execute()

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result)

	// Verify expectations
	mockProvider.AssertExpectations(t)
}
