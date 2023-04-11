package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func BuildWorkflowDistribution(workflowPath string) error {

	fmt.Println("Finding dependencies for workflow: ", workflowPath)
	dependencies, err := getDependenciesFromWdlContent2(workflowPath)
	if err != nil {
		return nil
	}

	fmt.Println("Packing dependencies into a zip file: ", dependencies)
	depsName := strings.Replace(filepath.Base(workflowPath), ".wdl", ".zip", 1)
	err = packDependencies(depsName, dependencies)
	if err != nil {
		return err
	}
	return nil
}

func getDependenciesFromWdlContent2(filePath string) ([]string, error) {
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
		subDependencies, err := getDependenciesFromWdlContent2(fullPath)
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
	// create a slice with basenames of files
	var filesToZip []string
	for _, file := range files {
		filesToZip = append(filesToZip, filepath.Base(file))
	}

	// check if any duplicated value in filesToZip
	if hasDuplicates(filesToZip) {
		return fmt.Errorf("duplicate files found in dependencies folder")
	}

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
	for _, filename := range files {
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

		// Set the name of the file within the zip archive
		header.Name = filepath.Base(filename)

		// Add the file to the zip archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		if err != nil {
			return err
		}
	}

	fmt.Println("Zip file created successfully")
	return nil
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
	for k, v := range occurs {
		if v > 1 {
			dup = true
			fmt.Println("duplicate files found in dependencies folder: ", k)
		}
	}
	return dup
}

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

func resolvePath(basePath, relativePath string) (string, error) {
	// Get the directory of the base path
	dir := filepath.Dir(basePath)

	// Join the directory with the relative path
	fullPath := filepath.Join(dir, relativePath)

	// Clean up the path to remove any redundant separators and dots
	fullPath = filepath.Clean(fullPath)

	return fullPath, nil
}
