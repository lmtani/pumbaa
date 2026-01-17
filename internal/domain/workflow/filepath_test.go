package workflow

import "testing"

func TestFilePath_IsGCS(t *testing.T) {
	tests := []struct {
		input    FilePath
		expected bool
	}{
		{"gs://bucket/file.txt", true},
		{"gs://bucket/path/to/file.bam", true},
		{"s3://bucket/file.txt", false},
		{"/path/to/file.txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.IsGCS() != tt.expected {
				t.Errorf("FilePath(%q).IsGCS() = %v, expected %v", tt.input, tt.input.IsGCS(), tt.expected)
			}
		})
	}
}

func TestFilePath_IsS3(t *testing.T) {
	tests := []struct {
		input    FilePath
		expected bool
	}{
		{"s3://bucket/file.txt", true},
		{"s3://bucket/path/to/file.bam", true},
		{"gs://bucket/file.txt", false},
		{"/path/to/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.IsS3() != tt.expected {
				t.Errorf("FilePath(%q).IsS3() = %v, expected %v", tt.input, tt.input.IsS3(), tt.expected)
			}
		})
	}
}

func TestFilePath_IsLocal(t *testing.T) {
	tests := []struct {
		input    FilePath
		expected bool
	}{
		{"/path/to/file.txt", true},
		{"/absolute/path/file.bam", true},
		{"/no-extension", false},
		{"relative/path.txt", false},
		{"gs://bucket/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.IsLocal() != tt.expected {
				t.Errorf("FilePath(%q).IsLocal() = %v, expected %v", tt.input, tt.input.IsLocal(), tt.expected)
			}
		})
	}
}

func TestFilePath_IsValid(t *testing.T) {
	tests := []struct {
		input    FilePath
		expected bool
	}{
		{"gs://bucket/file.txt", true},
		{"s3://bucket/file.txt", true},
		{"/path/to/file.txt", true},
		{"sample_name", false},
		{"echo hello", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if tt.input.IsValid() != tt.expected {
				t.Errorf("FilePath(%q).IsValid() = %v, expected %v", tt.input, tt.input.IsValid(), tt.expected)
			}
		})
	}
}

func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected int
	}{
		{"nil value", nil, 0},
		{"empty map", map[string]interface{}{}, 0},
		{"single GCS path", map[string]interface{}{"file": "gs://bucket/file.txt"}, 1},
		{"array of GCS paths", map[string]interface{}{
			"files": []interface{}{"gs://bucket/file1.txt", "gs://bucket/file2.txt"},
		}, 2},
		{"nested map", map[string]interface{}{
			"ref": map[string]interface{}{
				"fasta": "gs://bucket/ref.fasta",
				"index": "gs://bucket/ref.fasta.fai",
			},
		}, 2},
		{"mixed values", map[string]interface{}{
			"name":  "sample1",
			"count": 42,
			"file":  "gs://bucket/input.bam",
		}, 1},
		{"non-path strings", map[string]interface{}{
			"name":    "sample1",
			"command": "echo hello",
		}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFilePaths(tt.value)
			if len(result) != tt.expected {
				t.Errorf("ExtractFilePaths() returned %d paths, expected %d", len(result), tt.expected)
			}
		})
	}
}
