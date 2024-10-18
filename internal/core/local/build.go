package local

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lmtani/pumbaa/internal/share/informativeMessage"

	"github.com/fatih/color"
	"github.com/lmtani/pumbaa/internal/ports"
)

type Builder struct {
	wdl ports.Wdl
	fs  ports.Filesystem
}

func NewBuilder(wdl ports.Wdl, fs ports.Filesystem) *Builder {
	return &Builder{wdl: wdl, fs: fs}
}

func (r *Builder) findDependencies(wdlPath string) ([]string, error) {
	wdlContent, err := r.fs.ReadFile(wdlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WDL file %s: %w", wdlPath, err)
	}

	dependencies, err := r.wdl.GetDependencies(wdlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies for %s: %w", wdlPath, err)
	}

	for i, dep := range dependencies {
		dependencies[i] = resolvePath(wdlPath, dep)
	}

	var allDependencies []string
	for _, dep := range dependencies {
		allDependencies = append(allDependencies, dep)
		depDependencies, err := r.findDependencies(dep)
		if err != nil {
			return nil, fmt.Errorf("failed to find dependencies for %s: %w", dep, err)
		}
		allDependencies = append(allDependencies, depDependencies...)
	}

	return allDependencies, nil
}

// PackDependencies builds a zip file with all dependencies.
func (r *Builder) PackDependencies(workflowPath, outDir string) error {
	filesToZip, err := r.findDependencies(workflowPath)
	if err != nil {
		return err
	}
	if len(filesToZip) == 0 {
		// just the workflow, nothing to pack
		// so, just write in the output directory
		_, err := r.ReplaceImportsAndWrite(workflowPath, outDir)
		if err != nil {
			return fmt.Errorf("failed to replace imports and write: %w", err)
		}
		return nil
	}

	filesToZip, err = getUniqueFiles(filesToZip)
	if err != nil {
		return errors.New("duplicate files found in dependencies folder")
	}

	informativeMessage.InformativeMessage(color.FgGreen, "Dependencies for "+workflowPath)

	wdlDist, err := r.ReplaceImportsAndWrite(workflowPath, outDir)
	if err != nil {
		return fmt.Errorf("failed to replace imports and write: %w", err)
	}
	sort.Strings(filesToZip)
	toZip := []string{}
	for _, file := range filesToZip {
		fmt.Println("  -", file)
		tempDir, err := os.MkdirTemp("", "pumbaa-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}

		wdlDep, err := r.ReplaceImportsAndWrite(file, tempDir)
		if err != nil {
			return fmt.Errorf("failed to replace imports and write for dependency: %w", err)
		}
		toZip = append(toZip, wdlDep)
	}

	zipFileName := strings.TrimSuffix(filepath.Base(workflowPath), filepath.Ext(workflowPath)) + ".zip"
	zipPath := filepath.Join(outDir, zipFileName)

	err = r.fs.CreateZip(zipPath, toZip)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}

	fmt.Println("\nNow you can submit the workflow with something like:")
	informativeMessage.InformativeMessage(color.FgHiMagenta, "pumbaa submit --wdl "+wdlDist+" --dependencies "+zipPath+" <...>")
	return nil
}

func (r *Builder) ReplaceImportsAndWrite(workflowPath, outDir string) (string, error) {
	wdlContent, err := r.fs.ReadFile(workflowPath)
	if err != nil {
		return "", fmt.Errorf("failed to read WDL file %s: %w", workflowPath, err)
	}

	releaseWorkflow, err := r.wdl.ReplaceImports(wdlContent)
	if err != nil {
		return "", fmt.Errorf("failed to replace imports: %w", err)
	}

	err = r.fs.CreateDirectory(outDir)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	output := filepath.Join(outDir, filepath.Base(workflowPath))
	err = r.fs.WriteFile(output, releaseWorkflow)
	if err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", output, err)
	}
	return output, nil
}

func getUniqueFiles(toZip []string) ([]string, error) {
	occurs := make(map[string]string)
	var uniqueFiles []string

	for _, file := range toZip {
		fileName := filepath.Base(file)
		checksum, err := computeMD5(file)
		if err != nil {
			return nil, fmt.Errorf("error computing MD5 for file %s: %v", file, err)
		}
		if existingChecksum, exists := occurs[fileName]; exists {
			if existingChecksum != checksum {
				return nil, fmt.Errorf("duplicate files found with different checksums: %s", fileName)
			}
		} else {
			occurs[fileName] = checksum
			uniqueFiles = append(uniqueFiles, file)
		}
	}
	return uniqueFiles, nil
}

func resolvePath(basePath, relativePath string) string {
	return filepath.Join(filepath.Dir(basePath), relativePath)
}

func computeMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
