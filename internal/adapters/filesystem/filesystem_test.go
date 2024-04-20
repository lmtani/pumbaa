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
				t.Errorf("Error removing created file %s: %v", tt.args.srcPath, err)
			}
		})
	}
}

func TestLocalFilesystem_ReadFile(t *testing.T) {
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
			name: "ReadFile",
			fields: fields{
				l: nil,
			},
			args: args{
				path: "/tmp/test.txt",
			},
			want:    "Hello, World!",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			// write file with want value
			err := os.WriteFile(tt.args.path, []byte(tt.want), 0644)
			if err != nil {
				t.Errorf("Error writing file %s: %v", tt.args.path, err)
			}

			got, err := l.ReadFile(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalFilesystem_WriteFile(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		path     string
		contents string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "WriteFile",
			fields: fields{
				l: nil,
			},
			args: args{
				path:     "/tmp/test.txt",
				contents: "Hello, World!",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			if err := l.WriteFile(tt.args.path, tt.args.contents); (err != nil) != tt.wantErr {
				t.Errorf("WriteFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Read the file to check if the contents are correct
			got, err := l.ReadFile(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.args.contents {
				t.Errorf("ReadFile() got = %v, want %v", got, tt.args.contents)
			}

			// Cleanup
			err = os.RemoveAll(tt.args.path)
			if err != nil {
				t.Errorf("Error removing created file %s: %v", tt.args.path, err)
			}
		})
	}
}

func TestLocalFilesystem_CreateZip(t *testing.T) {
	type fields struct {
		l ports.Logger
	}
	type args struct {
		destinationPath string
		filePaths       []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "CreateZip",
			fields: fields{
				l: nil,
			},
			args: args{
				destinationPath: "/tmp/test.zip",
				filePaths: []string{
					"/tmp/test1.txt",
					"/tmp/test2.txt",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalFilesystem{
				l: tt.fields.l,
			}
			// Create files
			for _, path := range tt.args.filePaths {
				// write file without value
				err := os.WriteFile(path, []byte(""), 0644)
				if err != nil {
					t.Errorf("Error writing file %s: %v", path, err)
				}
			}

			if err := l.CreateZip(tt.args.destinationPath, tt.args.filePaths); (err != nil) != tt.wantErr {
				t.Errorf("CreateZip() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check if the zip file was created
			if _, err := os.Stat(tt.args.destinationPath); os.IsNotExist(err) {
				t.Errorf("Zip file not created: %v", err)
			}

			// Cleanup
			for _, path := range tt.args.filePaths {
				err := os.RemoveAll(path)
				if err != nil {
					t.Errorf("Error removing created file %s: %v", path, err)
				}
			}
		})
	}
}
