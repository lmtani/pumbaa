package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowGCEUsage_Execute_Success(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowGCEUsage(mockCromwell)

	input := &WorkflowGCEUsageInputDTO{
		WorkflowID: "test-workflow-id",
	}

	// We'll create two calls:
	// 1) Preemptible = "1", Disk = "local-disk 20 SSD", BootDiskSizeGb = "10", CPU="2", Memory="8 GB"
	//    Start= Jan 1, 2023, 00:00:00 => End= Jan 1, 2023, 03:00:00 => 3 hours
	// 2) Preemptible = "0", Disk = "local-disk 10 HDD", BootDiskSizeGb="5", CPU="1", Memory="4 GB"
	//    Start= Jan 1, 2023, 00:00:00 => End= Jan 1, 2023, 01:00:00 => 1 hour
	// We also show how a "cache hit" would skip usage aggregation, but we won't demonstrate that here.

	start1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end1 := start1.Add(3 * time.Hour) // 3 hours
	start2 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end2 := start2.Add(1 * time.Hour) // 1 hour

	calls := entities.CallItemSet{
		"someTask": {
			{
				ExecutionStatus: "Done",
				Start:           start1,
				End:             end1,
				RuntimeAttributes: entities.RuntimeAttributes{
					Preemptible:    "1",
					Disks:          "local-disk 20 SSD",
					BootDiskSizeGb: "10",
					CPU:            "2",
					Memory:         "8 GB",
				},
				CallCaching: entities.CallCachingData{Hit: false},
			},
			{
				ExecutionStatus: "Done",
				Start:           start2,
				End:             end2,
				RuntimeAttributes: entities.RuntimeAttributes{
					Preemptible:    "0",
					Disks:          "local-disk 10 HDD",
					BootDiskSizeGb: "5",
					CPU:            "1",
					Memory:         "4 GB",
				},
				CallCaching: entities.CallCachingData{Hit: false},
			},
		},
	}

	// Summarize expected usage:
	// For the first call (preemptible):
	//   - total disk = 20 + 10 = 30, it's SSD => PreemptSsd usage
	//   - 3 hours => totalSsd usage = 30 * 3 / 720 = 0.125
	//   - CPU=2 => 2 * 3 = 6 preempt CPU hours
	//   - Mem=8 => 8 * 3 = 24 preempt GB-hours
	// For the second call (non-preemptible):
	//   - total disk = 10 + 5 = 15, it's HDD => Hdd usage
	//   - 1 hour => totalHdd = 15 * 1 / 720 = 0.0208333...
	//   - CPU=1 => 1 * 1 = 1 CPU-hour
	//   - Mem=4 => 4 * 1 = 4 GB-hours
	// TotalTime= 3h + 1h = 4h

	// Put it in a MetadataResponse with status = "Succeeded"
	metadata := entities.MetadataResponse{
		Status: "Succeeded",
		Calls:  calls,
	}

	// Mock expectation
	mockCromwell.
		On("Metadata", "test-workflow-id", &entities.ParamsMetadataGet{ExpandSubWorkflows: true}).
		Return(metadata, nil)

	// WHEN
	output, err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err, "Error should be nil on success")
	assert.NotNil(t, output, "Output should not be nil on success")

	// Check some numeric fields with a small delta for floating inaccuracies
	// Preemptible usage
	assert.InDeltaf(t, 0.125, output.PreemptSsd, 0.0001, "PreemptSsd should match expected value")
	assert.InDeltaf(t, 24.0, output.PreemptMemory, 0.0001, "PreemptMemory should match expected value")
	assert.InDeltaf(t, 6.0, output.PreemptCPU, 0.0001, "PreemptCPU should match expected value")
	// Non-preemptible usage
	assert.InDeltaf(t, 0.0208333, output.Hdd, 0.0001, "Hdd usage should match expected value")
	assert.InDeltaf(t, 4.0, output.Memory, 0.0001, "Memory usage should match expected value")
	assert.InDeltaf(t, 1.0, output.CPU, 0.0001, "CPU usage should match expected value")
	// Summed times
	expectedTotal := 4 * time.Hour
	assert.Equal(t, expectedTotal, output.TotalTime, "Total time should be sum of 3h and 1h")
	// No cache hits
	assert.Equal(t, 0, output.CachedCalls, "No cached calls in this scenario")
}

func TestWorkflowGCEUsage_Execute_Running(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowGCEUsage(mockCromwell)

	input := &WorkflowGCEUsageInputDTO{
		WorkflowID: "test-workflow-id",
	}

	// If the workflow is still running, the use case returns an empty output
	// without error.
	metadata := entities.MetadataResponse{
		Status: "Running",
		Calls:  entities.CallItemSet{}, // calls might be empty
	}

	mockCromwell.
		On("Metadata", "test-workflow-id", &entities.ParamsMetadataGet{ExpandSubWorkflows: true}).
		Return(metadata, nil)

	// WHEN
	output, err := useCase.Execute(input)

	// THEN
	assert.NoError(t, err, "Should not be an error if status is Running")
	assert.NotNil(t, output, "Output should not be nil")
	// Everything should be zeroed
	assert.Equal(t, 0.0, output.PreemptHdd)
	assert.Equal(t, 0.0, output.PreemptSsd)
	assert.Equal(t, 0.0, output.PreemptCPU)
	assert.Equal(t, 0.0, output.PreemptMemory)
	assert.Equal(t, 0.0, output.Hdd)
	assert.Equal(t, 0.0, output.Ssd)
	assert.Equal(t, 0.0, output.CPU)
	assert.Equal(t, 0.0, output.Memory)
	assert.Equal(t, 0, output.CachedCalls)
	assert.Equal(t, time.Duration(0), output.TotalTime)
}

func TestWorkflowGCEUsage_Execute_Error(t *testing.T) {
	// GIVEN
	mockCromwell := new(mockCromwellServer)
	useCase := NewWorkflowGCEUsage(mockCromwell)

	input := &WorkflowGCEUsageInputDTO{
		WorkflowID: "test-workflow-id",
	}

	expectedError := errors.New("failed to get metadata")

	mockCromwell.
		On("Metadata", "test-workflow-id", &entities.ParamsMetadataGet{ExpandSubWorkflows: true}).
		Return(entities.MetadataResponse{}, expectedError)

	// WHEN
	output, err := useCase.Execute(input)

	// THEN
	assert.Error(t, err, "We expect an error")
	assert.Nil(t, output, "Output should be nil when there's an error")
	assert.Equal(t, expectedError, err, "Error should match the expected error")
}
