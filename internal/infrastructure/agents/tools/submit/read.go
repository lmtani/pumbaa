// Package submit provides agent actions that help prepare a workflow
// submission: scaffolding an inputs template and preflighting it. Both read
// local files (the WDL, the inputs JSON) sandboxed to the working directory
// pumbaa was launched from, the same boundary as write_file.
package submit

import (
	"fmt"
	"os"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/localfs"
)

// readWorkingDirFile reads a file the agent was pointed at, refusing paths
// that escape the working directory.
func readWorkingDirFile(path string) ([]byte, error) {
	full, err := localfs.ResolveWorkingDirPath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", path, err)
	}
	return data, nil
}
