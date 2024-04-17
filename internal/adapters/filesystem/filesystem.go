package filesystem

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lmtani/pumbaa/internal/ports"
)

type LocalFilesystem struct {
	l ports.Logger
}

func NewLocalFilesystem(l ports.Logger) *LocalFilesystem {
	return &LocalFilesystem{l: l}
}

func (l *LocalFilesystem) CreateDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0750)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *LocalFilesystem) HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func (l *LocalFilesystem) MoveFile(srcPath, destPath string) error {
	return os.Rename(srcPath, destPath)
}

func (l *LocalFilesystem) ZipFiles(workflowPath, zipPath string, files []string) ([]string, error) {
	l.l.Info(fmt.Sprintf("Creating zip file: %s", zipPath))
	var filesToZip []string
	for _, file := range files {
		fmt.Println("Adding file to zip: ", file)
		filesToZip = append(filesToZip, filepath.Base(file))
	}
	if l.hasDuplicates(filesToZip) {
		return filesToZip, fmt.Errorf("duplicate files found in dependencies folder")
	}

	// Replace import statements
	var replacedFiles []string
	for _, file := range files {
		l.l.Info(fmt.Sprintf("Replacing imports in file: %s", file))
		l.l.Info(fmt.Sprintf("Workflow path: %s", workflowPath))
		resolvedPath, err := l.resolvePath(workflowPath, file)
		if err != nil {
			return filesToZip, err
		}
		tempFile, err := l.ReplaceImports(resolvedPath)
		if err != nil {
			return filesToZip, err
		}
		replacedFiles = append(replacedFiles, tempFile)
	}

	if len(replacedFiles) == 0 {
		return replacedFiles, nil
	}

	// Create a new zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return filesToZip, err
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			println(err)
		}
	}(zipFile)

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			println(err)
		}
	}(zipWriter)

	// Add files to the zip archive
	for _, filename := range replacedFiles {
		l.l.Info(fmt.Sprintf("Adding file to zip: %s", filename))
		err := l.addFileToZip(filename, zipWriter)
		if err != nil {
			return filesToZip, err
		}
	}
	return filesToZip, nil
}

func (l *LocalFilesystem) ReplaceImports(path string) (string, error) {

	file, err := os.Open(path)
	if err != nil {
		l.l.Error(err.Error())
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			println(err)
		}
	}(file)

	outputFile, err := os.CreateTemp("", fmt.Sprintf("%s_*", filepath.Base(path)))
	l.l.Info(fmt.Sprintf("Creating temp file: %s", outputFile.Name()))
	if err != nil {
		l.l.Error(err.Error())
		return "", err
	}
	defer func(outputFile *os.File) {
		err := outputFile.Close()
		if err != nil {
			l.l.Error(err.Error())
		}
	}(outputFile)

	importRegex := regexp.MustCompile(`import\s+["'].*\/(.+)["']`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains an import statement
		match := importRegex.FindStringSubmatch(line)
		if len(match) > 0 {
			// Get the filename from the import statement
			filename := match[1]

			// Update the line with the new import statement
			newLine := strings.ReplaceAll(line, match[0], fmt.Sprintf(`import %q`, filename))

			// Write the modified line to the output file
			_, err := outputFile.WriteString(newLine + "\n")
			if err != nil {
				l.l.Error(err.Error())
				return "", err
			}

			// Print the original and modified import statements
			l.l.Info(fmt.Sprintf("Original import: %s\nModified import: %s\n", line, newLine))
		} else {
			// Write the original line to the output file
			_, err := outputFile.WriteString(line + "\n")
			if err != nil {
				l.l.Error(err.Error())
				return "", err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		l.l.Error(err.Error())
		return "", err
	}
	return outputFile.Name(), nil
}

func (l *LocalFilesystem) addFileToZip(filename string, zipWriter *zip.Writer) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			println(err)
		}
	}(file)

	// Get the file information
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create a new file header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Remove the suffix pattern after "_" due to temp file name
	filename = strings.Replace(filename, filepath.Ext(filename), "", 1)

	// Set the name of the file within the zip archive
	header.Name = filepath.Base(filename) + ".wdl"

	// Add the file to the zip archive
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	if err != nil {
		return err
	}
	return nil
}

func (l *LocalFilesystem) hasDuplicates(toZip []string) bool {
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
	for k, v := range occurs {
		if v > 1 {
			dup = true
			l.l.Info(fmt.Sprintf("duplicate files found in dependencies folder: %s", k))
		}
	}
	return dup
}

func (l *LocalFilesystem) resolvePath(basePath, relativePath string) (string, error) {
	// Get the directory of the base path
	dir := filepath.Dir(basePath)
	l.l.Info(fmt.Sprintf(" - Base path: %s", dir))
	l.l.Info(fmt.Sprintf(" - Relative path: %s", relativePath))

	// Join the directory with the relative path
	fullPath := filepath.Join(dir, relativePath)
	l.l.Info(fmt.Sprintf(" - Full path: %s", fullPath))

	// Clean up the path to remove any redundant separators and dots
	fullPath = filepath.Clean(fullPath)
	l.l.Info(fmt.Sprintf(" - Cleaned path: %s", fullPath))

	return fullPath, nil
}

func (l *LocalFilesystem) IsInUserPath(path string) bool {
	// Check if the path is in the user's path
	_, err := os.Stat(path)
	return err == nil
}
