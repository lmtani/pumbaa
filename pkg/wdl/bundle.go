package wdl

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BundleOptions configures the bundle creation process
type BundleOptions struct {
	// IncludeMetadata includes a manifest.json file in the bundle
	IncludeMetadata bool
	// PreserveDirectoryStructure maintains the relative directory structure
	PreserveDirectoryStructure bool
	// FlattenImports flattens all imports to a single directory (requires rewriting imports)
	FlattenImports bool
}

// DefaultBundleOptions returns the default bundle options
func DefaultBundleOptions() BundleOptions {
	return BundleOptions{
		IncludeMetadata:            true,
		PreserveDirectoryStructure: true,
		FlattenImports:             false,
	}
}

// BundleMetadata contains information about the bundle
type BundleMetadata struct {
	Version      string    `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	MainWorkflow string    `json:"main_workflow"`
	WDLVersion   string    `json:"wdl_version"`
	Dependencies []string  `json:"dependencies"`
	TotalFiles   int       `json:"total_files"`
}

// Bundle represents a self-contained WDL bundle
type Bundle struct {
	MainWorkflow string            // Path to the main workflow
	Files        map[string][]byte // All files in the bundle (relative path -> content)
	Metadata     *BundleMetadata
	Graph        *DependencyGraph
}

// CreateBundle creates a ZIP bundle containing the main workflow and all its dependencies
func CreateBundle(mainWorkflow string, outputPath string) error {
	return CreateBundleWithOptions(mainWorkflow, outputPath, DefaultBundleOptions())
}

// CreateBundleWithOptions creates a bundle with custom options
func CreateBundleWithOptions(mainWorkflow string, outputPath string, opts BundleOptions) error {
	bundle, err := BuildBundle(mainWorkflow, opts)
	if err != nil {
		return err
	}

	return bundle.WriteZip(outputPath)
}

// BuildBundle builds a bundle in memory without writing to disk
func BuildBundle(mainWorkflow string, opts BundleOptions) (*Bundle, error) {
	// Parse and analyze dependencies
	graph, err := AnalyzeDependenciesFromFile(mainWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	bundle := &Bundle{
		MainWorkflow: mainWorkflow,
		Files:        make(map[string][]byte),
		Graph:        graph,
	}

	// Get the base directory for relative paths
	baseDir := filepath.Dir(graph.Root)

	// Add main workflow
	mainContent, err := os.ReadFile(graph.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to read main workflow: %w", err)
	}

	mainRelPath := filepath.Base(graph.Root)
	bundle.Files[mainRelPath] = mainContent

	// Add all dependencies
	for _, depPath := range graph.Imports {
		content, err := os.ReadFile(depPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read dependency %s: %w", depPath, err)
		}

		var relPath string
		if opts.PreserveDirectoryStructure {
			relPath, err = filepath.Rel(baseDir, depPath)
			if err != nil {
				// If we can't get relative path, use the filename
				relPath = filepath.Base(depPath)
			}
		} else {
			relPath = filepath.Base(depPath)
		}

		bundle.Files[relPath] = content
	}

	// Build metadata
	if opts.IncludeMetadata {
		wdlVersion := ""
		if rootNode := graph.Dependencies[graph.Root]; rootNode != nil && rootNode.Document != nil {
			wdlVersion = rootNode.Document.Version
		}

		deps := make([]string, 0, len(graph.Imports))
		for _, depPath := range graph.Imports {
			relPath, err := filepath.Rel(baseDir, depPath)
			if err != nil {
				relPath = filepath.Base(depPath)
			}
			deps = append(deps, relPath)
		}

		bundle.Metadata = &BundleMetadata{
			Version:      "1.0",
			CreatedAt:    time.Now().UTC(),
			MainWorkflow: mainRelPath,
			WDLVersion:   wdlVersion,
			Dependencies: deps,
			TotalFiles:   len(bundle.Files),
		}
	}

	return bundle, nil
}

// WriteZip writes the bundle to a ZIP file
func (b *Bundle) WriteZip(outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// Write all WDL files
	for relPath, content := range b.Files {
		err := writeToZip(zipWriter, relPath, content)
		if err != nil {
			return fmt.Errorf("failed to write %s to zip: %w", relPath, err)
		}
	}

	// Write metadata if present
	if b.Metadata != nil {
		metadataJSON, err := json.MarshalIndent(b.Metadata, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		err = writeToZip(zipWriter, "manifest.json", metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}
	}

	return nil
}

// writeToZip writes content to a zip archive
func writeToZip(zw *zip.Writer, filename string, content []byte) error {
	writer, err := zw.Create(filename)
	if err != nil {
		return err
	}

	_, err = writer.Write(content)
	return err
}

// ExtractBundle extracts a bundle ZIP to a directory
func ExtractBundle(zipPath string, outputDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		err := extractZipFile(file, outputDir)
		if err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive
func extractZipFile(file *zip.File, outputDir string) error {
	// Sanitize file path to prevent zip slip
	filePath := filepath.Join(outputDir, file.Name)
	if !isPathWithinDir(filePath, outputDir) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(filePath, os.ModePerm)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// isPathWithinDir checks if a path is within a directory (prevents zip slip)
func isPathWithinDir(filePath string, dir string) bool {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	return filepath.HasPrefix(absFilePath, absDir)
}

// GetMainWorkflowContent returns the content of the main workflow
func (b *Bundle) GetMainWorkflowContent() ([]byte, error) {
	mainRel := filepath.Base(b.MainWorkflow)
	if content, ok := b.Files[mainRel]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("main workflow not found in bundle")
}

// ListFiles returns all files in the bundle
func (b *Bundle) ListFiles() []string {
	files := make([]string, 0, len(b.Files))
	for path := range b.Files {
		files = append(files, path)
	}
	return files
}

// GetFile returns the content of a specific file in the bundle
func (b *Bundle) GetFile(path string) ([]byte, bool) {
	content, ok := b.Files[path]
	return content, ok
}
