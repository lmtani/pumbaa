package commands

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/lmtani/cromwell-cli/internal/prompt"
)

func ExampleCommands_Navigate() {
	// Mock http server
	content, err := ioutil.ReadFile("../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "")
	prompt := prompt.NewForTests()
	err = cmds.Navigate(prompt, operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Workflow: HelloHere
	//
	// Command status: Done
	// echo "HelloWorld!"
	// Logs:
	// gs://bucket/HelloHere/a1606e3a-611e-4a60-8cac-dcbe90ce3d14/call-SayGoodbye/stderr
	// gs://bucket/HelloHere/a1606e3a-611e-4a60-8cac-dcbe90ce3d14/call-SayGoodbye/stdout
	//
	// gs://bucket/HelloHere/a1606e3a-611e-4a60-8cac-dcbe90ce3d14/call-SayGoodbye/SayGoodbye.log
	//
	// 🐋 Docker image:
	// ubuntu:20.04
}
