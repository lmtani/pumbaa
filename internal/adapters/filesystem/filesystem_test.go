package filesystem

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

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
