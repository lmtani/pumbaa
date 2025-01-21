package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestWorkflowQueryUseCase_Execute_Success tests a successful query scenario.
func TestWorkflowQueryUseCase_Execute_Success(t *testing.T) {
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowQuery(mockCromwell)

	input := &WorkflowQueryInputDTO{
		Name: "my-test-wf",
		Days: 3,
	}

	// We don't have direct control over time.Now() in the code, so we either match the
	// "Submission" param loosely or skip verifying it exactly. We'll use a custom matcher
	// that checks the Name but doesn't strictly check the time.

	expectedResult := entities.QueryResponse{
		Results: []entities.QueryResponseWorkflow{
			{
				ID:                    "abc-123",
				Name:                  "my-test-wf",
				Status:                "Succeeded",
				Submission:            "2023-01-02T15:04:05Z",
				Start:                 time.Date(2023, 1, 2, 15, 4, 5, 0, time.UTC),
				End:                   time.Date(2023, 1, 2, 17, 4, 5, 0, time.UTC),
				MetadataArchiveStatus: "Archived",
			},
		},
		TotalResultsCount: 2,
	}

	// A custom matcher that checks only the Name field on ParamsQueryGet
	paramMatcher := mock.MatchedBy(func(p *entities.ParamsQueryGet) bool {
		return p.Name == "my-test-wf"
	})

	// Expectation: Query is called once, returns our expectedResult
	mockCromwell.
		On("Query", paramMatcher).
		Return(expectedResult, nil).
		Once()

	// WHEN
	output, err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err, "Should not return an error on success")
	assert.NotNil(t, output, "Output should not be nil")
	assert.Len(t, output.Workflows, 1, "We expect 2 workflows in the output")

	// Check a few fields from the first workflow
	w1 := output.Workflows[0]
	assert.Equal(t, "abc-123", w1.ID)
	assert.Equal(t, "my-test-wf", w1.Name)
	assert.Equal(t, "Succeeded", w1.Status)
	assert.Equal(t, "Archived", w1.MetadataArchiveStatus)

	// Verify the mock was called as expected
	mockCromwell.AssertExpectations(t)
}

// TestWorkflowQueryUseCase_Execute_Error tests when Cromwell's Query returns an error.
func TestWorkflowQueryUseCase_Execute_Error(t *testing.T) {
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowQuery(mockCromwell)

	input := &WorkflowQueryInputDTO{
		Name: "any-name",
		Days: 0,
	}

	expectedErr := errors.New("query failed")
	mockCromwell.
		On("Query", mock.AnythingOfType("*entities.ParamsQueryGet")).
		Return(entities.QueryResponse{}, expectedErr)

	output, err := useCase.Execute(input)

	assert.Error(t, err, "Expected an error from Query")
	assert.Nil(t, output, "Output should be nil when error occurs")
	assert.Equal(t, expectedErr, err, "Error should match the expected error")

	mockCromwell.AssertExpectations(t)
}
