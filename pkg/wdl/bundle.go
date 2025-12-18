package wdl

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// importRegex matches import statements: import "path" or import 'path'
var importRegex = regexp.MustCompile(`import\s+["']([^"']+)["']`)

// maxExtractFileSize limits the size of files extracted from ZIP (100MB)
const maxExtractFileSize = 100 * 1024 * 1024

// BundleOptions configures the bundle creation process
type BundleOptions struct {
	// IncludeMetadata includes a manifest.json file in the bundle
	IncludeMetadata bool
}

// DefaultBundleOptions returns the default bundle options
func DefaultBundleOptions() BundleOptions {
	return BundleOptions{
		IncludeMetadata: true,
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

// BundleResult contains the output of the bundle creation
type BundleResult struct {
	// MainWDLPath is the path to the generated main WDL file with rewritten imports
	MainWDLPath string
	// DependenciesZipPath is the path to the ZIP file containing all dependencies
	DependenciesZipPath string
	// Dependencies is the list of dependency paths included in the ZIP
	Dependencies []string
	// TotalFiles is the total number of files (main WDL + dependencies)
	TotalFiles int
}

// CreateBundle creates a WDL bundle for Cromwell submission.
// It produces:
// 1. A main WDL file with imports rewritten to reference files inside the ZIP
// 2. A ZIP file containing all dependencies with flattened structure
// The outputDir parameter specifies the directory where outputs will be created.
// Output filenames are derived from the mainWorkflow basename.
func CreateBundle(mainWorkflow string, outputDir string) (*BundleResult, error) {
	return CreateBundleWithOptions(mainWorkflow, outputDir, DefaultBundleOptions())
}

// CreateBundleWithOptions creates a bundle with custom options
func CreateBundleWithOptions(mainWorkflow string, outputDir string, opts BundleOptions) (*BundleResult, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse and analyze dependencies
	graph, err := AnalyzeDependenciesFromFile(mainWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Derive output filenames from main workflow basename
	baseName := strings.TrimSuffix(filepath.Base(graph.Root), filepath.Ext(graph.Root))
	mainWDLPath := filepath.Join(outputDir, baseName+".wdl")
	zipPath := filepath.Join(outputDir, baseName+".zip")

	// No dependencies - just copy the main workflow
	if len(graph.Imports) == 0 {
		mainContent, err := os.ReadFile(graph.Root)
		if err != nil {
			return nil, fmt.Errorf("failed to read main workflow: %w", err)
		}

		// Write main WDL without changes
		if err := os.WriteFile(mainWDLPath, mainContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to write main WDL: %w", err)
		}

		return &BundleResult{
			MainWDLPath:         mainWDLPath,
			DependenciesZipPath: "",
			Dependencies:        []string{},
			TotalFiles:          1,
		}, nil
	}

	// Build import mapping: original import path -> flattened name in ZIP
	importMapping := buildImportMapping(graph)

	// Rewrite main workflow imports
	mainContent, err := os.ReadFile(graph.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to read main workflow: %w", err)
	}

	rewrittenMain := rewriteImports(string(mainContent), graph.Root, importMapping)

	// Write rewritten main WDL
	if err := os.WriteFile(mainWDLPath, []byte(rewrittenMain), 0644); err != nil {
		return nil, fmt.Errorf("failed to write main WDL: %w", err)
	}

	// Create ZIP with dependencies
	if err := createDependenciesZip(graph, importMapping, zipPath, opts); err != nil {
		// Clean up main WDL on failure
		os.Remove(mainWDLPath)
		return nil, fmt.Errorf("failed to create dependencies ZIP: %w", err)
	}

	deps := append([]string{}, graph.Imports...)

	return &BundleResult{
		MainWDLPath:         mainWDLPath,
		DependenciesZipPath: zipPath,
		Dependencies:        deps,
		TotalFiles:          len(graph.Imports) + 1,
	}, nil
}

// buildImportMapping creates a mapping from absolute paths to flattened names for the ZIP
func buildImportMapping(graph *DependencyGraph) map[string]string {
	mapping := make(map[string]string)
	usedNames := make(map[string]int)

	for _, depPath := range graph.Imports {
		baseName := filepath.Base(depPath)

		// Handle name collisions
		if count, exists := usedNames[baseName]; exists {
			ext := filepath.Ext(baseName)
			nameWithoutExt := strings.TrimSuffix(baseName, ext)
			baseName = fmt.Sprintf("%s_%d%s", nameWithoutExt, count+1, ext)
			usedNames[filepath.Base(depPath)] = count + 1
		} else {
			usedNames[baseName] = 1
		}

		mapping[depPath] = baseName
	}

	return mapping
}

// rewriteImports rewrites import statements in WDL content
func rewriteImports(content string, filePath string, importMapping map[string]string) string {
	fileDir := filepath.Dir(filePath)

	return importRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := importRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		importPath := submatches[1]

		// Resolve to absolute path
		var absPath string
		if filepath.IsAbs(importPath) {
			absPath = importPath
		} else {
			absPath = filepath.Clean(filepath.Join(fileDir, importPath))
		}

		// Look up the flattened name
		if flatName, ok := importMapping[absPath]; ok {
			return fmt.Sprintf(`import "%s"`, flatName)
		}

		// Not found in mapping - keep original
		return match
	})
}

// createDependenciesZip creates a ZIP file containing all dependencies
func createDependenciesZip(graph *DependencyGraph, importMapping map[string]string, zipPath string, opts BundleOptions) (err error) {
	outFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to finalize zip: %w", closeErr)
		}
	}()

	// Write each dependency with rewritten imports
	for depPath, flatName := range importMapping {
		content, err := os.ReadFile(depPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", depPath, err)
		}

		// Rewrite imports in this dependency too
		rewrittenContent := rewriteImports(string(content), depPath, importMapping)

		if err := writeFileToZip(zipWriter, flatName, []byte(rewrittenContent)); err != nil {
			return fmt.Errorf("failed to write %s to zip: %w", flatName, err)
		}
	}

	// Write metadata if requested
	if opts.IncludeMetadata {
		wdlVersion := ""
		if rootNode := graph.Dependencies[graph.Root]; rootNode != nil && rootNode.Document != nil {
			wdlVersion = rootNode.Document.Version
		}

		deps := make([]string, 0, len(importMapping))
		for _, flatName := range importMapping {
			deps = append(deps, flatName)
		}

		metadata := &BundleMetadata{
			Version:      "1.0",
			CreatedAt:    time.Now().UTC(),
			MainWorkflow: filepath.Base(graph.Root),
			WDLVersion:   wdlVersion,
			Dependencies: deps,
			TotalFiles:   len(deps) + 1,
		}

		metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		if err := writeFileToZip(zipWriter, "manifest.json", metadataJSON); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}
	}

	return nil
}

// writeFileToZip writes a file to a ZIP archive with proper headers
func writeFileToZip(zw *zip.Writer, filename string, content []byte) error {
	header := &zip.FileHeader{
		Name:     filename,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = writer.Write(content)
	return err
}

// ExtractBundle extracts a bundle ZIP to a directory
// Used for testing and verification purposes
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

	// Limit extracted file size to prevent zip bombs
	_, err = io.CopyN(outFile, rc, maxExtractFileSize)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
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
	// Ensure we check with path separator to avoid prefix matching issues
	// e.g., /tmp/output vs /tmp/output-evil
	return strings.HasPrefix(absFilePath, absDir+string(os.PathSeparator)) || absFilePath == absDir
}
