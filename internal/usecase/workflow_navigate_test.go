package usecase

import (
	"errors"
	"github.com/lmtani/pumbaa/internal/interfaces"
	"testing"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockWriter mocks the entities.Writer interface.
type mockWriter struct {
	mock.Mock
}

func (m *mockWriter) Primary(msg string) {
	m.Called(msg)
}

func (m *mockWriter) Accent(msg string) {
	m.Called(msg)
}

func (m *mockWriter) Message(msg string) {
	m.Called(msg)
}

func (m *mockWriter) Error(msg string) {
	m.Called(msg)
}

func (m *mockWriter) Table(table interfaces.Table) {
	m.Called(table)
}

func (m *mockWriter) QueryTable(table entities.QueryResponse) {
	m.Called(table)
}

func (m *mockWriter) ResourceTable(table entities.TotalResources) {
	m.Called(table)
}

func (m *mockWriter) MetadataTable(d entities.MetadataResponse) error {
	args := m.Called(d)
	return args.Error(0)
}

func (m *mockWriter) Json(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}

// mockPrompt mocks the entities.Prompt interface.
type mockPrompt struct {
	mock.Mock
}

func (m *mockPrompt) SelectByKey(taskOptions []string) (string, error) {
	args := m.Called(taskOptions)
	return args.String(0), args.Error(1)
}

// We define a "searcher" function in the real code to compare the user's input with shard index.
// The library calls p.SelectByIndex(searcher, shards).
func (m *mockPrompt) SelectByIndex(searcher func(input string, index int) bool, items interface{}) (int, error) {
	// Because we can't pass the searcher function in mock calls easily, let's assume we return
	// an integer or an error directly. We'll rely on the test setup to specify the shard index
	// we want to simulate.
	args := m.Called(items)
	return args.Int(0), args.Error(1)
}

// ---------------- Tests ---------------- //

func TestWorkflowNavigate_Execute_Success_SingleShard_NoSubworkflow(t *testing.T) {
	// Scenario:
	// - Workflow has no subworkflow.
	// - The user selects a single task named "taskOne".
	// - That task has exactly one shard => no shard prompt is called.
	// - The item is not queued, not a cache hit, so we should Accent the command line.

	// Mocks
	mCrom := new(mockCromwellServer)
	mWriter := new(mockWriter)
	mPrompt := new(mockPrompt)

	useCase := NewWorkflowNavigate(mCrom, mWriter, mPrompt)

	input := &WorkflowNavigateInputDTO{WorkflowID: "my-workflow"}

	// The calls structure has a single entry: "wf.taskOne" -> slice with a single shard
	calls := entities.CallItemSet{
		"wf.taskOne": {
			{
				ExecutionStatus: "Done",
				CommandLine:     "echo Hello World",
				ShardIndex:      0,
				CallCaching:     entities.CallCachingData{Hit: false, Result: ""},
				// No SubWorkflowID => not a subworkflow
			},
		},
	}
	meta := entities.MetadataResponse{
		WorkflowName: "my-workflow",
		Calls:        calls,
	}

	expectedParams := &entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}

	// Expectation: Cromwell Metadata call
	mCrom.On("Metadata", "my-workflow", expectedParams).Return(meta, nil)

	// Expectation: Prompt to select "taskOne"
	// -> We return "taskOne"
	mPrompt.On("SelectByKey", []string{"taskOne"}).Return("taskOne", nil)

	// Because there's only one shard in calls["taskOne"], selectByIndex won't be called.

	// We'll just match them in any order we like.
	mWriter.On("Accent", "Workflow: my-workflow\n").Return()
	mWriter.On("Accent", "Done").Return()
	mWriter.On("Accent", "echo Hello World").Return()

	// WHEN
	err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err)
	mCrom.AssertExpectations(t)
	mPrompt.AssertExpectations(t)
	mWriter.AssertExpectations(t)
}

func TestWorkflowNavigate_Execute_Success_Subworkflow(t *testing.T) {
	// Scenario:
	// - The top-level metadata has a single call with SubWorkflowID => we call metadata again.
	// - The sub-workflow has multiple tasks. The user selects "taskTwo".
	// - taskTwo has 2 shards => user is prompted to pick a shard by index.

	// Mocks
	mCrom := new(mockCromwellServer)
	mWriter := new(mockWriter)
	mPrompt := new(mockPrompt)

	useCase := NewWorkflowNavigate(mCrom, mWriter, mPrompt)

	input := &WorkflowNavigateInputDTO{WorkflowID: "main-workflow"}

	// Top-level call referencing a subworkflow
	topLevelCalls := entities.CallItemSet{
		"wf.subworkflowCall": {
			{
				ExecutionStatus: "Done",
				SubWorkflowID:   "subworkflow-id",
				ShardIndex:      0,
			},
		},
	}
	topMeta := entities.MetadataResponse{
		WorkflowName: "MainWF",
		Calls:        topLevelCalls,
	}

	// Subworkflow calls
	subCalls := entities.CallItemSet{
		"sw.taskOne": {
			{ShardIndex: 0, ExecutionStatus: "Done"},
		},
		"sw.taskTwo": {
			{ShardIndex: 0, ExecutionStatus: "QueuedInCromwell"},
			{ShardIndex: 1, ExecutionStatus: "Done", CallCaching: entities.CallCachingData{Hit: true, Result: "Cache Hit Result"}},
		},
	}
	subMeta := entities.MetadataResponse{
		WorkflowName:   "SubWF",
		RootWorkflowID: "main-workflow", // indicates subworkflow
		Calls:          subCalls,
	}

	expectedParams := &entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}

	// Set up the mock calls
	// 1) First metadata => top-level
	mCrom.On("Metadata", "main-workflow", expectedParams).Return(topMeta, nil)
	// 2) Because the single item has SubWorkflowID, we call metadata again
	mCrom.On("Metadata", "subworkflow-id", expectedParams).Return(subMeta, nil)

	// Prompt usage:
	// At the top-level, we only have "wf.subworkflowCall", which yields a single "taskName" = "subworkflowCall".
	// So we ask SelectByKey -> returns "subworkflowCall"
	// Then we skip shard selection because there's only 1 call item in that top-level array, but it has SubWorkflowID => go deeper
	// Then for the subworkflow, tasks are "taskOne" and "taskTwo". Prompt user chooses "taskTwo".
	// Then we have multiple shards in "taskTwo", so we call SelectByIndex. We'll simulate user picking shard 1.

	mPrompt.On("SelectByKey", []string{"subworkflowCall"}).Return("subworkflowCall", nil).Once()
	// Now in the subworkflow, we have two tasks: "taskOne" and "taskTwo"
	mPrompt.On("SelectByKey", []string{"taskOne", "taskTwo"}).Return("taskTwo", nil).Once()

	// "taskTwo" has 2 shards => we call SelectByIndex(...).
	// We'll say the user picks index=1 (the second shard).
	mPrompt.On("SelectByIndex", mock.Anything, mock.Anything).Return(1, nil).Once()

	// Writer calls:
	// For top-level:
	//   Accent("Workflow: MainWF\n")
	// For subworkflow:
	//   Accent("SubWorkflow: SubWF\n")
	//
	// Then final item: shard index=1, ExecutionStatus= "Done", call caching hit =>
	//   Accent("Done")
	//   Accent("Cache Hit Result")
	mWriter.On("Accent", "Workflow: MainWF\n").Return().Once()
	mWriter.On("Accent", "SubWorkflow: SubWF\n").Return().Once()
	mWriter.On("Accent", "Done").Return().Once()
	// Because call caching is a hit, we accent the caching result, not the command line
	mWriter.On("Accent", "Cache Hit Result").Return().Once()

	// WHEN
	err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err)
	mCrom.AssertExpectations(t)
	mPrompt.AssertExpectations(t)
	mWriter.AssertExpectations(t)
}

func TestWorkflowNavigate_Execute_MetadataError(t *testing.T) {
	// Scenario: Cromwell returns an error on the first metadata call => the use case fails.

	mCrom := new(mockCromwellServer)
	mWriter := new(mockWriter)
	mPrompt := new(mockPrompt)

	useCase := NewWorkflowNavigate(mCrom, mWriter, mPrompt)

	input := &WorkflowNavigateInputDTO{WorkflowID: "broken-workflow"}

	expectedErr := errors.New("metadata retrieval failed")
	expectedParams := &entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}

	mCrom.On("Metadata", "broken-workflow", expectedParams).Return(entities.MetadataResponse{}, expectedErr)

	err := useCase.Execute(input)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	mCrom.AssertExpectations(t)
	// No writer/prompt calls expected
}

func TestWorkflowNavigate_Execute_PromptSelectByKeyError(t *testing.T) {
	// Scenario: The metadata works, but the user prompt fails at selectDesiredTask.
	// We'll test that we get the same error back from the use case.

	mCrom := new(mockCromwellServer)
	mWriter := new(mockWriter)
	mPrompt := new(mockPrompt)

	useCase := NewWorkflowNavigate(mCrom, mWriter, mPrompt)

	input := &WorkflowNavigateInputDTO{WorkflowID: "some-workflow"}

	calls := entities.CallItemSet{
		"wf.taskA": {{ExecutionStatus: "Done"}},
	}
	meta := entities.MetadataResponse{
		WorkflowName: "TestWorkflow",
		Calls:        calls,
	}
	expectedParams := &entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}

	mCrom.On("Metadata", "some-workflow", expectedParams).Return(meta, nil)

	promptErr := errors.New("prompt failure")
	// We'll pass the single task name "taskA" to the prompt, but we simulate error
	mPrompt.On("SelectByKey", []string{"taskA"}).Return("", promptErr)

	// We'll expect an Accent call for "Workflow: TestWorkflow\n"
	mWriter.On("Accent", "Workflow: TestWorkflow\n").Return()

	err := useCase.Execute(input)
	assert.Error(t, err)
	assert.Equal(t, promptErr, err)

	mCrom.AssertExpectations(t)
	mPrompt.AssertExpectations(t)
	mWriter.AssertExpectations(t)
}

func TestWorkflowNavigate_Execute_PromptSelectByIndexError(t *testing.T) {
	// Scenario: The user selected a task that has multiple shards => the prompt fails at selectByIndex.

	mCrom := new(mockCromwellServer)
	mWriter := new(mockWriter)
	mPrompt := new(mockPrompt)

	useCase := NewWorkflowNavigate(mCrom, mWriter, mPrompt)

	input := &WorkflowNavigateInputDTO{WorkflowID: "some-workflow"}

	calls := entities.CallItemSet{
		"wf.taskB": {
			{ShardIndex: 0, ExecutionStatus: "Done"},
			{ShardIndex: 1, ExecutionStatus: "Failed"},
		},
	}
	meta := entities.MetadataResponse{
		WorkflowName: "TestMultiShard",
		Calls:        calls,
	}
	expectedParams := &entities.ParamsMetadataGet{
		ExcludeKey: []string{"executionEvents", "submittedFiles", "jes", "inputs"},
	}

	mCrom.On("Metadata", "some-workflow", expectedParams).Return(meta, nil)

	// The user picks "taskB"
	mPrompt.On("SelectByKey", []string{"taskB"}).Return("taskB", nil)

	promptErr := errors.New("shard selection error")
	// The code passes shards to SelectByIndex; we simulate returning an error
	mPrompt.On("SelectByIndex", mock.Anything, mock.Anything).Return(0, promptErr)

	// Expect accent for the workflow info
	mWriter.On("Accent", "Workflow: TestMultiShard\n").Return()

	err := useCase.Execute(input)
	assert.Error(t, err)
	assert.Equal(t, promptErr, err)

	mCrom.AssertExpectations(t)
	mPrompt.AssertExpectations(t)
	mWriter.AssertExpectations(t)
}
