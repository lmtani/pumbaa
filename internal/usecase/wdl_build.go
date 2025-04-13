// To many use of external packages, this usecase is not tested.
// If you have suggestions on how to test this usecase or better
// organize, please let me know.

package usecase

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/lmtani/pumbaa/internal/interfaces"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type BuildInputDTO struct {
	WDLPath string
	OutPath string
}

type WDLBuilder struct {
	FileSystem interfaces.Filesystem
	WDL        interfaces.Wdl
}

func NewWDLBuilder(fs interfaces.Filesystem, wdl interfaces.Wdl) *WDLBuilder {
	return &WDLBuilder{FileSystem: fs, WDL: wdl}
}

// Execute builds a zip file with all dependencies.
func (b *WDLBuilder) Execute(input BuildInputDTO) error {
	filesToZip, err := b.findDependencies(input.WDLPath)
	if err != nil {
		return err
	}
	if len(filesToZip) == 0 {
		// Cenario when there are no dependencies, just copy the main WDL to the output directory
		_, err := b.ReplaceImportsAndWrite(input.WDLPath, input.OutPath)
		if err != nil {
			return fmt.Errorf("failed to replace imports and write: %w", err)
		}
		return nil
	}

	filesToZip, err = getUniqueFiles(filesToZip)
	if err != nil {
		return errors.New("duplicate files found in dependencies folder")
	}

	fmt.Println("Dependencies for " + input.WDLPath)

	wdlDist, err := b.ReplaceImportsAndWrite(input.WDLPath, input.OutPath)
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

		wdlDep, err := b.ReplaceImportsAndWrite(file, tempDir)
		if err != nil {
			return fmt.Errorf("failed to replace imports and write for dependency: %w", err)
		}
		toZip = append(toZip, wdlDep)
	}

	zipFileName := strings.TrimSuffix(filepath.Base(input.WDLPath), filepath.Ext(input.WDLPath)) + ".zip"
	zipPath := filepath.Join(input.OutPath, zipFileName)

	err = b.FileSystem.CreateZip(zipPath, toZip)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}

	fmt.Println("\nNow you can submit the workflow with something like:")
	fmt.Println("pumbaa submit --wdl " + wdlDist + " --dependencies " + zipPath + " <...>")
	return nil
}

func (b *WDLBuilder) findDependencies(wdlPath string) ([]string, error) {
	wdlContent, err := b.FileSystem.ReadFile(wdlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read WDL file %s: %w", wdlPath, err)
	}

	dependencies, err := b.WDL.GetDependencies(wdlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies for %s: %w", wdlPath, err)
	}

	for i, dep := range dependencies {
		dependencies[i] = resolvePath(wdlPath, dep)
	}

	var allDependencies []string
	for _, dep := range dependencies {
		allDependencies = append(allDependencies, dep)
		depDependencies, err := b.findDependencies(dep)
		if err != nil {
			return nil, fmt.Errorf("failed to find dependencies for %s: %w", dep, err)
		}
		allDependencies = append(allDependencies, depDependencies...)
	}

	return allDependencies, nil
}

func (b *WDLBuilder) ReplaceImportsAndWrite(wdl_path, outDir string) (string, error) {
	wdlContent, err := b.FileSystem.ReadFile(wdl_path)
	if err != nil {
		return "", fmt.Errorf("failed to read WDL file %s: %w", wdl_path, err)
	}

	releaseWorkflow, err := b.WDL.ReplaceImports(wdlContent)
	if err != nil {
		return "", fmt.Errorf("failed to replace imports: %w", err)
	}

	err = b.FileSystem.CreateDirectory(outDir)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory %s: %w", outDir, err)
	}

	output := filepath.Join(outDir, filepath.Base(wdl_path))
	err = b.FileSystem.WriteFile(output, releaseWorkflow)
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
