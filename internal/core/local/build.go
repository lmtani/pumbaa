package local

import (
	"fmt"
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

// WorkflowDist It builds a zip file with all dependencies.
// It also produces a new WDL file to remove relative imports.
func (r *Builder) WorkflowDist(workflowPath, outDir string) error {
	matches, err := r.wdl.GetDependencies(workflowPath)
	if err != nil {
		return err
	}

	releaseWorkflow, err := r.fs.ReplaceImports(workflowPath)
	if err != nil {
		return err
	}

	err = r.fs.CreateDirectory(outDir)
	if err != nil {
		return err
	}

	newName := strings.Replace(releaseWorkflow, filepath.Ext(releaseWorkflow), "", 1)
	newName = filepath.Base(newName) + ".wdl"
	// TODO: maybe refactor to:
	// newName := filepath.Base(strings.TrimSuffix(releaseWorkflow, filepath.Ext(releaseWorkflow))) + ".wdl"
	err = r.fs.MoveFile(releaseWorkflow, filepath.Join(outDir, newName))
	if err != nil {
		return err
	}

	depNames := strings.Replace(filepath.Base(workflowPath), ".wdl", ".zip", 1)
	dependencies := r.secondElementOfListOfLists(matches)
	zipName, err := r.fs.ZipFiles(workflowPath, depNames, dependencies)
	if err != nil {
		return err
	}
	fmt.Println("Moving file to releases directory: ", zipName)
	return nil
}

func (r *Builder) secondElementOfListOfLists(lol [][]string) []string {
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
