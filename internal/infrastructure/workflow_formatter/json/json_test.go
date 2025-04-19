package json

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowJsonFormatter_Query(t *testing.T) {
	// Arrange
	buffer := new(bytes.Buffer)
	formatter := NewWorkflowJsonFormatter(buffer)

	now := time.Now()
	workflows := []entities.Workflow{
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

	// Act
	err := formatter.Query(workflows)

	// Assert
	assert.NoError(t, err)

	// Decode the JSON to verify it contains the expected data
	var decodedWorkflows []entities.Workflow
	err = json.Unmarshal(buffer.Bytes(), &decodedWorkflows)

	assert.NoError(t, err)
	assert.Len(t, decodedWorkflows, 2)
	assert.Equal(t, "workflow-1", decodedWorkflows[0].ID)
	assert.Equal(t, "Test Workflow 1", decodedWorkflows[0].Name)
	assert.Equal(t, "Succeeded", decodedWorkflows[0].Status)
	assert.Equal(t, "workflow-2", decodedWorkflows[1].ID)
	assert.Equal(t, "Test Workflow 2", decodedWorkflows[1].Name)
	assert.Equal(t, "Running", decodedWorkflows[1].Status)
}

func TestWorkflowJsonFormatter_Report(t *testing.T) {
	// Arrange
	buffer := new(bytes.Buffer)
	formatter := NewWorkflowJsonFormatter(buffer)

	now := time.Now()
	workflow := entities.Workflow{
		ID:     "workflow-3",
		Name:   "Test Workflow 3",
		Start:  now,
		End:    now.Add(time.Hour * 3),
		Status: "Failed",
		Calls:  map[string][]entities.Step{},
	}

	// Act
	err := formatter.Report(&workflow)

	// Assert
	assert.NoError(t, err)

	// Decode the JSON to verify it contains the expected data
	var decodedWorkflow entities.Workflow
	err = json.Unmarshal(buffer.Bytes(), &decodedWorkflow)

	assert.NoError(t, err)
	assert.Equal(t, "workflow-3", decodedWorkflow.ID)
	assert.Equal(t, "Test Workflow 3", decodedWorkflow.Name)
	assert.Equal(t, "Failed", decodedWorkflow.Status)
}

func TestWorkflowJsonFormatter_DefaultOutput(t *testing.T) {
	// Test that the formatter uses stdout as default output when nil is provided
	formatter := NewWorkflowJsonFormatter(nil)
	assert.NotNil(t, formatter.Output, "Output should default to stdout when nil is provided")
}
