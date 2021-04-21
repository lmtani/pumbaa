package commands

import (
	"log"
)

func ExampleSubmitWorkflow() {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1", `{"id": "a-new-uuid", "status": "Submitted"}`)
	defer ts.Close()

	wdlPath := "../sample/wf.wdl"
	inputsPath := "../sample/wf.inputs.json"
	err := SubmitWorkflow(ts.URL, "", wdlPath, inputsPath, wdlPath, inputsPath)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// 🐖 Operation= a-new-uuid , Status=Submitted
}
