package wdl

import (
	"reflect"
	"testing"
)

func TestRegexWdlPArser_GetDependencies(t *testing.T) {
	type args struct {
		contents string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{name: "Test 1", args: args{contents: `import "file1.wdl"`}, want: []string{"file1.wdl"}, wantErr: false},
		{name: "Test 2", args: args{contents: `import "../../task.wdl"`}, want: []string{"../../task.wdl"}, wantErr: false},
		{name: "Test 3", args: args{contents: `import "file1.wdl"\nimport "file2.wdl"`}, want: []string{"file1.wdl", "file2.wdl"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RegexWdlPArser{}
			got, err := r.GetDependencies(tt.args.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDependencies() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegexWdlPArser_ReplaceImports(t *testing.T) {
	type args struct {
		contents string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "Test 1", args: args{contents: `import "file1.wdl"`}, want: `import "file1.wdl"` + "\n", wantErr: false},
		{name: "Test 2", args: args{contents: `import "../../task.wdl"`}, want: `import "task.wdl"` + "\n", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RegexWdlPArser{}
			got, err := r.ReplaceImports(tt.args.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceImports() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReplaceImports() got = %v, want %v", got, tt.want)
			}
		})
	}
}
