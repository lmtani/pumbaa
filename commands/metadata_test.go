package commands

import (
	"fmt"
	"io/ioutil"
	"log"
)

func ExampleCommands_MetadataWorkflow() {
	// Read metadata mock
	content, err := ioutil.ReadFile("../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "")
	err = cmds.MetadataWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// +---------------+---------+----------+--------+
	// |     TASK      | ATTEMPT | ELAPSED  | STATUS |
	// +---------------+---------+----------+--------+
	// | SayGoodbye    | 1       | 720h0m0s | Done   |
	// | SayHello      | 1       | 720h0m0s | Done   |
	// | SayHelloCache | 1       | 720h0m0s | Done   |
	// +---------------+---------+----------+--------+
}
