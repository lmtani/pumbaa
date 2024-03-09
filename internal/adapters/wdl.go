package adapters

import (
	"os"
	"regexp"
)

type RegexWdlPArser struct{}

func (r *RegexWdlPArser) GetDependencies(workflowPath string) ([][]string, error) {
	// Load the content of the file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, err
	}

	// Define a regular expression to match import statements
	re := regexp.MustCompile(`import\s+["'](.+?)["']`)

	// Find all import paths and store them in a slice of strings
	return re.FindAllStringSubmatch(string(content), -1), nil
}
