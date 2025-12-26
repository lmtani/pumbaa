package bundle

import (
	"errors"
	"testing"
)

func TestBundle_Initialization(t *testing.T) {
	// Although simple structs, we can verify initialization logic
	// if we decide to add constructor functions in the future.
	// For now, we test the basic structure.

	b := Bundle{
		MainWorkflow: "main.wdl",
		Files: map[string][]byte{
			"main.wdl": []byte("workflow w {}"),
		},
		Metadata: &Metadata{
			Version:    "1.0.0",
			WDLVersion: "1.0",
		},
	}

	if b.MainWorkflow != "main.wdl" {
		t.Errorf("got %s, want main.wdl", b.MainWorkflow)
	}

	if len(b.Files) != 1 {
		t.Errorf("got %d files, want 1", len(b.Files))
	}
}

func TestDependencyError_Error(t *testing.T) {
	inner := errors.New("file not found")
	tests := []struct {
		name string
		err  DependencyError
		want string
	}{
		{
			name: "without cause",
			err:  DependencyError{Path: "lib.wdl", Message: "missing"},
			want: "dependency error for 'lib.wdl': missing",
		},
		{
			name: "with cause",
			err:  DependencyError{Path: "lib.wdl", Message: "failed", Cause: inner},
			want: "dependency error for 'lib.wdl': failed (caused by: file not found)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("DependencyError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDependencyError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := DependencyError{Cause: inner}
	if got := err.Unwrap(); got != inner {
		t.Errorf("DependencyError.Unwrap() = %v, want %v", got, inner)
	}
}
