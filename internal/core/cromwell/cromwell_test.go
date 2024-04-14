package cromwell

import (
	"os"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/adapters/test"
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

func TestCromwell_Outputs(t *testing.T) {
	type fields struct {
		s ports.CromwellServer
		w ports.Writer
	}
	type args struct {
		o string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Outputs",
			fields: fields{
				s: &test.FakeCromwell{
					OutputsResponse: types.OutputsResponse{
						ID: "operation",
						Outputs: map[string]interface{}{
							"output": "output",
						},
					},
				},
				w: adapters.NewColoredWriter(os.Stdout),
			},
			args: args{
				o: "operation",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
				w: tt.fields.w,
			}
			if err := c.Outputs(tt.args.o); (err != nil) != tt.wantErr {
				t.Errorf("Outputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCromwell_QueryWorkflow(t *testing.T) {
	type fields struct {
		s ports.CromwellServer
		w ports.Writer
	}
	type args struct {
		name string
		days time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test QueryWorkflow",
			fields: fields{
				s: &test.FakeCromwell{
					QueryResponse: types.QueryResponse{
						Results: []types.QueryResponseWorkflow{
							{
								ID:                    "operation",
								Name:                  "TestWorkflow",
								Status:                "status",
								Submission:            "submission",
								Start:                 time.Now(),
								End:                   time.Now(),
								MetadataArchiveStatus: "metadata",
							},
						},
					},
				},
				w: adapters.NewColoredWriter(os.Stdout),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
				w: tt.fields.w,
			}
			if err := c.QueryWorkflow(tt.args.name, tt.args.days); (err != nil) != tt.wantErr {
				t.Errorf("QueryWorkflow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
