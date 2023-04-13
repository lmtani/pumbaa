package util

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BuildWorkflowDist It builds a zip file with all dependencies.
func BuildWorkflowDist(workflowPath string) error {

	fmt.Println("Finding dependencies for workflow: ", workflowPath)
	dependencies, err := getDependencies(workflowPath)
	if err != nil {
		return nil
	}

	// Modify WDL file to have simplified imports. i.e.: import "dependencies.wdl" instead of import "path/to/dependencies.wdl"
	releaseWorkflow, err := replaceImports(workflowPath)
	if err != nil {
		return err
	}

	// create releases directory
	err = createDirectory("releases")
	if err != nil {
		return err
	}
	// set write permission to releases directory
	err = os.Chmod("releases", 0777)
	if err != nil {
		return err
	}

	// remove suffix pattern. Ex: aaa_vvv_ddd.wdl_a23143123 -> aaa_vvv_ddd.wdl
	newName := strings.Replace(releaseWorkflow, filepath.Ext(releaseWorkflow), "", 1)
	// get filename
	newName = filepath.Base(newName) + ".wdl"

	// move the modified WDL file to the releases directory
	fmt.Println("Moving file to releases directory: ", newName)
	err = os.Rename(releaseWorkflow, filepath.Join("releases", newName))
	if err != nil {
		return err
	}

	depsName := strings.Replace(filepath.Base(workflowPath), ".wdl", ".zip", 1)
	err = packDependencies(depsName, dependencies)
	if err != nil {
		return err
	}
	// move the zip file to the releases directory
	err = os.Rename(depsName, filepath.Join("releases", depsName))
	if err != nil {
		return err
	}
	return nil
}

func replaceImports(path string) (string, error) {

	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer file.Close()

	outputFile, err := os.CreateTemp("", fmt.Sprintf("%s_*", filepath.Base(path)))
	fmt.Println("Creating temp file: ", outputFile.Name())
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer outputFile.Close()

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
			newLine := strings.ReplaceAll(line, match[0], fmt.Sprintf(`import "%s"`, filename))

			// Write the modified line to the output file
			_, err := outputFile.WriteString(newLine + "\n")
			if err != nil {
				fmt.Println(err)
				return "", err
			}

			// Print the original and modified import statements
			fmt.Printf("Original import: %s\nModified import: %s\n", line, newLine)
		} else {
			// Write the original line to the output file
			_, err := outputFile.WriteString(line + "\n")
			if err != nil {
				fmt.Println(err)
				return "", err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		return "", err
	}
	return outputFile.Name(), nil
}

// getDependencies It recursively finds all dependencies.
func getDependencies(filePath string) ([]string, error) {
	var importPaths []string
	// Load the content of the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("entrou no erro", err)
		return nil, err
	}

	// Define a regular expression to match import statements
	re := regexp.MustCompile(`import\s+["'](.+?)["']`)

	// Find all import paths and store them in a slice of strings
	matches := re.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		fullPath, err := resolvePath(filePath, match[1])
		importPaths = append(importPaths, fullPath)
		if err != nil {
			return nil, err
		}
		subDependencies, err := getDependencies(fullPath)
		if err != nil {
			fmt.Println("entrou no erro", err)
			return nil, err
		}

		importPaths = append(importPaths, subDependencies...)
	}
	// remove duplicates
	importPaths = removeDuplicates(importPaths)

	return importPaths, nil
}

// packDependencies packs all files in a folder into a zip file
// It will return an error if there are duplicated files in the folder
func packDependencies(n string, files []string) error {
	// create a slice with basenames of files and check if any duplicated value in filesToZip
	var filesToZip []string
	for _, file := range files {
		filesToZip = append(filesToZip, filepath.Base(file))
	}
	if hasDuplicates(filesToZip) {
		return fmt.Errorf("duplicate files found in dependencies folder")
	}

	// Replace import statements
	var replacedFiles []string
	for _, file := range files {
		tempFile, err := replaceImports(file)
		if err != nil {
			return err
		}
		replacedFiles = append(replacedFiles, tempFile)
	}
	fmt.Println(replacedFiles)

	// Create a new zip file
	zipFile, err := os.Create(n)
	if err != nil {

		return err
	}
	defer zipFile.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add files to the zip archive
	for _, filename := range replacedFiles {
		fmt.Println("Adding file to zip: ", filename)
		err := addFileToZip(filename, zipWriter)
		if err != nil {
			return err
		}
	}

	fmt.Println("Zip file created successfully")
	return nil
}

func addFileToZip(filename string, zipWriter *zip.Writer) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

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

// hasDuplicates checks if there are duplicated values in a slice
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
	for k, v := range occurs {
		if v > 1 {
			dup = true
			fmt.Println("duplicate files found in dependencies folder: ", k)
		}
	}
	return dup
}

// removeDuplicates removes duplicated values from a slice
func removeDuplicates(input []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, val := range input {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			result = append(result, val)
		}
	}

	return result
}

// resolvePath resolves a relative path to an absolute path
func resolvePath(basePath, relativePath string) (string, error) {
	// Get the directory of the base path
	dir := filepath.Dir(basePath)

	// Join the directory with the relative path
	fullPath := filepath.Join(dir, relativePath)

	// Clean up the path to remove any redundant separators and dots
	fullPath = filepath.Clean(fullPath)

	return fullPath, nil
}
