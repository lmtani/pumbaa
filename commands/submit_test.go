package commands

import (
	"log"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

func ExampleCommands_SubmitWorkflow() {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1", `{"id": "a-new-uuid", "status": "Submitted"}`)
	defer ts.Close()

	wdlPath := "../sample/wf.wdl"
	inputsPath := "../sample/wf.inputs.json"

	c := cromwell.New(ts.URL, "")
	cmds := New(c)
	err := cmds.SubmitWorkflow(wdlPath, inputsPath, wdlPath, inputsPath)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// 🐖 Operation= a-new-uuid , Status=Submitted
}
