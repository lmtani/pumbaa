// Package workflow contains the domain entities and business logic for workflows.
package workflow

import (
	"strings"
)

// FilePath is a Value Object representing a file path.
// It encapsulates validation and categorization of file paths (GCS, S3, local).
type FilePath string

// IsGCS returns true if the path is a Google Cloud Storage path.
func (p FilePath) IsGCS() bool {
	return strings.HasPrefix(string(p), "gs://")
}

// IsS3 returns true if the path is an Amazon S3 path.
func (p FilePath) IsS3() bool {
	return strings.HasPrefix(string(p), "s3://")
}

// IsLocal returns true if the path is a local absolute path with a file extension.
func (p FilePath) IsLocal() bool {
	s := string(p)
	return strings.HasPrefix(s, "/") && strings.Contains(s, ".")
}

// IsValid returns true if the path is a recognized file path format.
func (p FilePath) IsValid() bool {
	return p.IsGCS() || p.IsS3() || p.IsLocal()
}

// String returns the original path string.
func (p FilePath) String() string {
	return string(p)
}

// ExtractFilePaths recursively extracts all valid file paths from a value.
// It handles strings, slices, and maps.
func ExtractFilePaths(value any) []FilePath {
	var paths []FilePath
	extractFilePathsRecursive(value, &paths)
	return paths
}

func extractFilePathsRecursive(value any, paths *[]FilePath) {
	switch v := value.(type) {
	case string:
		p := FilePath(v)
		if p.IsValid() {
			*paths = append(*paths, p)
		}
	case []any:
		for _, item := range v {
			extractFilePathsRecursive(item, paths)
		}
	case map[string]any:
		for _, val := range v {
			extractFilePathsRecursive(val, paths)
		}
	}
}
