package wdl

import (
	"fmt"
	"regexp"
	"strings"
)

type RegexWdlPArser struct{}

func (r *RegexWdlPArser) GetDependencies(contents string) ([]string, error) {
	// Define a regular expression to match import statements
	re := regexp.MustCompile(`import\s+["'](.+?)["']`)

	// Find all import paths and store them in a slice of strings
	matches := r.secondElementOfListOfLists(re.FindAllStringSubmatch(contents, -1))
	return matches, nil
}

func (r *RegexWdlPArser) ReplaceImports(contents string) (string, error) {
	importRegex := regexp.MustCompile(`import\s+["'].*\/(.+)["']`)
	var builder strings.Builder

	lines := strings.Split(contents, "\n")
	for _, line := range lines {

		// Check if the line contains an import statement
		match := importRegex.FindStringSubmatch(line)
		if len(match) > 0 {
			// Get the filename from the import statement
			filename := match[1]

			// Update the line with the new import statement
			newLine := strings.ReplaceAll(line, match[0], fmt.Sprintf(`import %q`, filename))
			if _, err := builder.WriteString(newLine + "\n"); err != nil {
				return "", err
			}

		} else {
			// Write the original line to the output file
			if _, err := builder.WriteString(line + "\n"); err != nil {
				return "", err
			}
		}
	}

	return builder.String(), nil
}

func (r *RegexWdlPArser) secondElementOfListOfLists(lol [][]string) []string {
	if len(lol) == 0 {
		return nil
	}
	var secondElements []string
	for _, l := range lol {
		fmt.Println("Second element of list of lists: ", l[1])
		secondElements = append(secondElements, l[1])
	}
	return secondElements
}
