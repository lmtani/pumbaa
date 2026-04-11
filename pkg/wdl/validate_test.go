package wdl

import (
	"errors"
	"testing"
)

func TestValidateInputs(t *testing.T) {
	tests := []struct {
		name        string
		wdl         string
		inputs      string
		wantErr     bool
		wantMissing []string
	}{
		{
			name: "all required inputs provided",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
        File reference
    }
}`,
			inputs:  `{"Hello.name": "world", "Hello.reference": "gs://bucket/ref.fa"}`,
			wantErr: false,
		},
		{
			name: "missing required input",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
        File reference
    }
}`,
			inputs:      `{"Hello.name": "world"}`,
			wantErr:     true,
			wantMissing: []string{"Hello.reference"},
		},
		{
			name: "missing all required inputs with empty JSON",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
        File reference
    }
}`,
			inputs:      `{}`,
			wantErr:     true,
			wantMissing: []string{"Hello.name", "Hello.reference"},
		},
		{
			name: "missing all required inputs with nil inputs",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
    }
}`,
			inputs:      "",
			wantErr:     true,
			wantMissing: []string{"Hello.name"},
		},
		{
			name: "optional inputs not required",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
        String? greeting
        Int? max_retries
    }
}`,
			inputs:  `{"Hello.name": "world"}`,
			wantErr: false,
		},
		{
			name: "inputs with defaults not required",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
        String greeting = "hi"
        Int max_retries = 3
    }
}`,
			inputs:  `{"Hello.name": "world"}`,
			wantErr: false,
		},
		{
			name: "no required inputs",
			wdl: `version 1.0
workflow Hello {
    input {
        String? name
        String greeting = "hi"
    }
}`,
			inputs:  `{}`,
			wantErr: false,
		},
		{
			name: "workflow without inputs section",
			wdl: `version 1.0
workflow Hello {
    output {
        String msg = "hello"
    }
}`,
			inputs:  `{}`,
			wantErr: false,
		},
		{
			name: "no workflow in WDL skips validation",
			wdl: `version 1.0
task Only {
    command { echo "hi" }
    output { String out = read_string(stdout()) }
}`,
			inputs:  `{}`,
			wantErr: false,
		},
		{
			name: "invalid WDL skips validation",
			wdl:    `not valid wdl`,
			inputs:  `{}`,
			wantErr: false,
		},
		{
			name: "invalid JSON returns error",
			wdl: `version 1.0
workflow Hello {
    input {
        String name
    }
}`,
			inputs:  `{invalid`,
			wantErr: true,
		},
		{
			name: "complex types as required",
			wdl: `version 1.0
workflow Pipeline {
    input {
        Array[File] samples
        Map[String, String] metadata
        File reference
        Array[String]? optional_flags
    }
}`,
			inputs:      `{"Pipeline.reference": "gs://ref.fa"}`,
			wantErr:     true,
			wantMissing: []string{"Pipeline.samples", "Pipeline.metadata"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputs([]byte(tt.wdl), []byte(tt.inputs))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantMissing != nil {
					var missingErr *MissingInputsError
					if !errors.As(err, &missingErr) {
						t.Fatalf("expected MissingInputsError, got %T: %v", err, err)
					}
					if len(missingErr.Missing) != len(tt.wantMissing) {
						t.Fatalf("expected %d missing inputs, got %d: %v", len(tt.wantMissing), len(missingErr.Missing), missingErr.Missing)
					}
					got := make(map[string]bool)
					for _, m := range missingErr.Missing {
						got[m] = true
					}
					for _, want := range tt.wantMissing {
						if !got[want] {
							t.Errorf("expected missing input %q, got %v", want, missingErr.Missing)
						}
					}
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
