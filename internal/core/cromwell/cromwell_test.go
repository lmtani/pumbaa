package cromwell

import (
	"os"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/adapters/writer"

	"github.com/lmtani/pumbaa/internal/adapters/test"
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

var outputsResponse = types.OutputsResponse{
	ID: "operation",
	Outputs: map[string]interface{}{
		"output": "output",
	},
}

var queryResponse = types.QueryResponse{
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
}

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
				s: &test.FakeCromwell{OutputsResponse: outputsResponse},
				w: writer.NewColoredWriter(os.Stdout),
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
			}
			data, err := c.Outputs(tt.args.o)
			if err != nil {
				t.Errorf("Outputs() error = %v", err)
			}
			if data.ID != tt.fields.s.(*test.FakeCromwell).OutputsResponse.ID {
				t.Errorf("Outputs() = %v, want %v", data.ID, tt.fields.s.(*test.FakeCromwell).OutputsResponse.ID)
			}
			if data.Outputs["output"] != tt.fields.s.(*test.FakeCromwell).OutputsResponse.Outputs["output"] {
				t.Errorf("Outputs() = %v, want %v", data.Outputs["output"], tt.fields.s.(*test.FakeCromwell).OutputsResponse.Outputs["output"])
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
				s: &test.FakeCromwell{QueryResponse: queryResponse},
				w: writer.NewColoredWriter(os.Stdout),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
			}
			data, err := c.QueryWorkflow(tt.args.name, tt.args.days)
			if err != nil {
				t.Errorf("QueryWorkflow() error = %v", err)
			}
			if data.Results[0].ID != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].ID {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].ID, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].ID)
			}
			if data.Results[0].Name != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Name {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].Name, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Name)
			}
			if data.Results[0].Status != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Status {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].Status, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Status)
			}
			if data.Results[0].Submission != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Submission {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].Submission, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Submission)
			}
			if data.Results[0].Start != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Start {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].Start, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].Start)
			}
			if data.Results[0].End != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].End {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].End, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].End)
			}
			if data.Results[0].MetadataArchiveStatus != tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].MetadataArchiveStatus {
				t.Errorf("QueryWorkflow() = %v, want %v", data.Results[0].MetadataArchiveStatus, tt.fields.s.(*test.FakeCromwell).QueryResponse.Results[0].MetadataArchiveStatus)
			}
		})
	}
}
