package local

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/lmtani/pumbaa/internal/ports"
)

type Builder struct {
	wdl ports.Wdl
	fs  ports.Filesystem
}

func NewBuilder(wdl ports.Wdl, fs ports.Filesystem) *Builder {
	return &Builder{wdl: wdl, fs: fs}
}

// PackDependencies It builds a zip file with all dependencies.
// It also produces a new WDL file to remove relative imports.
func (r *Builder) PackDependencies(workflowPath, outDir string) error {
	wdlContent, err := r.ReplaceImportsAndWrite(workflowPath, outDir)
	if err != nil {
		return err
	}

	dependencies, err := r.wdl.GetDependencies(wdlContent)
	if err != nil {
		return err
	}
	if len(dependencies) == 0 {
		return nil
	}

	var filesToZip []string
	for _, dependency := range dependencies {
		path := resolvePath(workflowPath, dependency)
		_, err = r.ReplaceImportsAndWrite(path, outDir)
		if err != nil {
			return err
		}
		outPath := filepath.Join(outDir, filepath.Base(dependency))
		filesToZip = append(filesToZip, outPath)
	}
	if hasDuplicates(filesToZip) {
		return errors.New("duplicate files found in dependencies folder")
	}

	// Ensure the filename change only affects the extension
	baseName := filepath.Base(workflowPath)
	zipFileName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".zip"

	// Build the full output path for the zip file
	zipPath := filepath.Join(outDir, zipFileName)

	return r.fs.CreateZip(zipPath, filesToZip)
}

func (r *Builder) ReplaceImportsAndWrite(workflowPath string, outDir string) (string, error) {
	wdlContent, err := r.fs.ReadFile(workflowPath)
	if err != nil {
		return "", err
	}

	releaseWorkflow, err := r.wdl.ReplaceImports(wdlContent)
	if err != nil {
		return "", err
	}

	err = r.fs.CreateDirectory(outDir)
	if err != nil {
		return "", err
	}

	err = r.fs.WriteFile(filepath.Join(outDir, filepath.Base(workflowPath)), releaseWorkflow)
	if err != nil {
		return "", err
	}
	return wdlContent, nil
}

func hasDuplicates(toZip []string) bool {
	// create a map to store the count of each element
	// in the slice
	occurs := make(map[string]int)

	// iterate over the slice and store the count of
	// each element in the map
	for _, num := range toZip {
		occurs[num]++
	}

	// iterate over the map and check if any element
	// has a count greater than 1
	dup := false
	for _, v := range occurs {
		if v > 1 {
			dup = true
		}
	}
	return dup
}

func resolvePath(basePath, relativePath string) string {
	// Get the directory of the base path
	dir := filepath.Dir(basePath)

	// Join the directory with the relative path
	fullPath := filepath.Join(dir, relativePath)

	// Clean up the path to remove any redundant separators and dots
	fullPath = filepath.Clean(fullPath)

	return fullPath
}
