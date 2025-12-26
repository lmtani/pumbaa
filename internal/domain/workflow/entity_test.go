package workflow

import (
	"testing"
	"time"
)

func TestWorkflow_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"Succeeded is terminal", StatusSucceeded, true},
		{"Failed is terminal", StatusFailed, true},
		{"Aborted is terminal", StatusAborted, true},
		{"Running is not terminal", StatusRunning, false},
		{"Submitted is not terminal", StatusSubmitted, false},
		{"Aborting is not terminal", StatusAborting, false},
		{"On Hold is not terminal", StatusOnHold, false},
		{"Unknown is not terminal", StatusUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workflow{Status: tt.status}
			if got := w.IsTerminal(); got != tt.want {
				t.Errorf("Workflow.IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorkflow_Duration(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	tests := []struct {
		name     string
		workflow *Workflow
		want     time.Duration
		approx   bool
	}{
		{
			name:     "Zero start time returns zero duration",
			workflow: &Workflow{},
			want:     0,
		},
		{
			name: "Start and end time set",
			workflow: &Workflow{
				Start: oneHourAgo,
				End:   now,
			},
			want: time.Hour,
		},
		{
			name: "Start set, end not set (running)",
			workflow: &Workflow{
				Start: oneHourAgo,
			},
			want:   time.Hour,
			approx: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.workflow.Duration()
			if tt.approx {
				// Allow 1 second difference for running workflows
				diff := got - tt.want
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Second {
					t.Errorf("Workflow.Duration() = %v, want approx %v", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("Workflow.Duration() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{Field: "ID", Message: "required"}
	want := "validation error on field 'ID': required"
	if got := err.Error(); got != want {
		t.Errorf("ValidationError.Error() = %q, want %q", got, want)
	}
}

func TestAPIError_Error(t *testing.T) {
	err := APIError{StatusCode: 404, Message: "Not Found"}
	want := "API error (status 404): Not Found"
	if got := err.Error(); got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}
