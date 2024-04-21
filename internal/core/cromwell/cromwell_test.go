package cromwell

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/adapters/logger"

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

func TestCromwell_Metadata(t *testing.T) {
	type fields struct {
		s ports.CromwellServer
		l ports.Logger
	}
	type args struct {
		o string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    types.MetadataResponse
		wantErr bool
	}{
		{
			name: "Test Metadata",
			fields: fields{
				s: &test.FakeCromwell{MetadataResponse: types.MetadataResponse{
					WorkflowName: "Teste",
					Inputs: map[string]interface{}{
						"input": "input",
					},
				}},
				l: logger.NewLogger(logger.InfoLevel),
			},
			args: args{
				o: "operation",
			},
			want: types.MetadataResponse{
				WorkflowName: "Teste",
				Inputs: map[string]interface{}{
					"input": "input",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
				l: tt.fields.l,
			}
			got, err := c.Metadata(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("Metadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Metadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func loadMetadataJson() types.MetadataResponse {
	// Load a json file in project root and return the type
	// internal/adapters/cromwellclient/testdata/metadata.json
	// load file
	content, err := os.ReadFile("../../adapters/cromwellclient/testdata/metadata.json")
	if err != nil {
		panic(err)
	}
	// load content into metadataresponse
	var metadata types.MetadataResponse
	err = json.Unmarshal(content, &metadata)
	if err != nil {
		panic(err)
	}
	return metadata
}

func TestCromwell_ResourceUsages(t *testing.T) {
	type fields struct {
		s ports.CromwellServer
		l ports.Logger
	}
	type args struct {
		o string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    types.TotalResources
		wantErr bool
	}{
		{
			name: "Test ResourceUsages",
			fields: fields{
				s: &test.FakeCromwell{MetadataResponse: loadMetadataJson()},
				l: logger.NewLogger(logger.InfoLevel),
			},
			args: args{
				o: "operation",
			},
			want: types.TotalResources{
				PreemptHdd:    20,
				PreemptMemory: 2880,
				PreemptSsd:    20,
				PreemptCPU:    1440,
				Hdd:           0,
				Memory:        1440,
				CPU:           720,
				Ssd:           20,
				TotalTime:     7776000000000000,
				CachedCalls:   1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
				l: tt.fields.l,
			}
			got, err := c.ResourceUsages(tt.args.o)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourceUsages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceUsages() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCromwell_SubmitWorkflow(t *testing.T) {
	type fields struct {
		s ports.CromwellServer
		l ports.Logger
	}
	type args struct {
		wdl          string
		inputs       string
		dependencies string
		options      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    types.SubmitResponse
		wantErr bool
	}{
		{
			name: "Test SubmitWorkflow",
			fields: fields{
				s: &test.FakeCromwell{SubmitResponse: types.SubmitResponse{
					ID: "operation",
				}},
				l: logger.NewLogger(logger.InfoLevel),
			},
			args: args{
				wdl:          "wdl",
				inputs:       "inputs",
				dependencies: "dependencies",
				options:      "options",
			},
			want: types.SubmitResponse{
				ID: "operation",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
				l: tt.fields.l,
			}
			got, err := c.SubmitWorkflow(tt.args.wdl, tt.args.inputs, tt.args.dependencies, tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubmitWorkflow() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCromwell_Wait(t *testing.T) {
	type fields struct {
		s ports.CromwellServer
		l ports.Logger
	}
	type args struct {
		operation string
		sleep     int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Wait",
			fields: fields{
				s: &test.FakeCromwell{SubmitResponse: types.SubmitResponse{
					ID:     "operation",
					Status: "Succeeded",
				}},
				l: logger.NewLogger(logger.InfoLevel),
			},
			args: args{
				operation: "operation",
				sleep:     1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cromwell{
				s: tt.fields.s,
				l: tt.fields.l,
			}
			if err := c.Wait(tt.args.operation, tt.args.sleep); (err != nil) != tt.wantErr {
				t.Errorf("Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
