package filesystem

import (
	"archive/zip"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/adapters/logger"
	"github.com/lmtani/pumbaa/internal/ports"
)

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func TestLocalFilesystem_CreateDirectory(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "CreateDirectory",
			fields: fields{
				l: nil,
			},
			args: args{
				dir: fmt.Sprintf("/tmp/test_%s", randomString(10)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			if err := l.CreateDirectory(tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("CreateDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Cleanup
			err := os.RemoveAll(tt.args.dir)
			if err != nil {
				t.Errorf("Error removing directory %s: %v", tt.args.dir, err)
			}
		})
	}
}

func TestLocalFilesystem_HomeDir(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "HomeDir",
			fields: fields{
				l: nil,
			},
			want:    os.Getenv("HOME"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			got, err := l.HomeDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("HomeDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HomeDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//func TestLocalFilesystem_IsInUserPath(t *testing.T) {
//	type fields struct {
//		l ports.Logger
//	}
//	type args struct {
//		path string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		{
//			name: "IsInUserPath",
//			fields: fields{
//				l: nil,
//			},
//			args: args{
//				path: os.Getenv("HOME"),
//			}
//		}
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			l := &LocalFilesystem{
//				l: tt.fields.l,
//			}
//			if got := l.IsInUserPath(tt.args.path); got != tt.want {
//				t.Errorf("IsInUserPath() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func TestLocalFilesystem_MoveFile(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		srcPath  string
		destPath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "MoveFile",
			fields: fields{
				l: nil,
			},
			args: args{
				srcPath:  fmt.Sprintf("/tmp/test_%s", randomString(10)),
				destPath: fmt.Sprintf("/tmp/test_%s", randomString(10)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a file
			_, err := os.Create(tt.args.srcPath)
			if err != nil {
				t.Errorf("Error creating file %s: %v", tt.args.srcPath, err)
			}

			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			if err := l.MoveFile(tt.args.srcPath, tt.args.destPath); (err != nil) != tt.wantErr {
				t.Errorf("MoveFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Cleanup
			err = os.RemoveAll(tt.args.srcPath)
			if err != nil {
				t.Errorf("Error removing directory %s: %v", tt.args.srcPath, err)
			}
		})
	}
}

func TestLocalFilesystem_ReplaceImports(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ReplaceImports",
			fields: fields{
				l: logger.NewLogger(logger.InfoLevel),
			},
			args: args{
				path: "../../../assets/workflow.wdl",
			},
			want:    "/tmp/workflow.wdl",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			got, err := l.ReplaceImports(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceImports() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("ReplaceImports() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalFilesystem_ZipFiles(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		workflowPath string
		zipPath      string
		files        []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			got, err := l.ZipFiles(tt.args.workflowPath, tt.args.zipPath, tt.args.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("ZipFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ZipFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalFilesystem_addFileToZip(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		filename  string
		zipWriter *zip.Writer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			if err := l.addFileToZip(tt.args.filename, tt.args.zipWriter); (err != nil) != tt.wantErr {
				t.Errorf("addFileToZip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLocalFilesystem_hasDuplicates(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		toZip []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			if got := l.hasDuplicates(tt.args.toZip); got != tt.want {
				t.Errorf("hasDuplicates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalFilesystem_resolvePath(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		basePath     string
		relativePath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			got, err := l.resolvePath(tt.args.basePath, tt.args.relativePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolvePath() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLocalFilesystem(t *testing.T) {
	type args struct {
		l ports.Logger
	}
	tests := []struct {
		name string
		args args
		want *LocalFilesystem
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLocalFilesystem(tt.args.l); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLocalFilesystem() = %v, want %v", got, tt.want)
			}
		})
	}
}
