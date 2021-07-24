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
	// +-------------------+---------+----------+--------+
	// |       TASK        | ATTEMPT | ELAPSED  | STATUS |
	// +-------------------+---------+----------+--------+
	// | RunHelloWorkflows | 1       | 7.515s   | Done   |
	// | RunHelloWorkflows | 1       | 7.514s   | Done   |
	// | SayGoodbye        | 1       | 720h0m0s | Done   |
	// | SayHello          | 1       | 720h0m0s | Done   |
	// | SayHelloCache     | 1       | 720h0m0s | Done   |
	// +-------------------+---------+----------+--------+
}

func ExampleCommands_MetadataWorkflow_second() {
	// Read metadata mock
	content, err := ioutil.ReadFile("../pkg/cromwell/mocks/metadata-failed.json")
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
	// +------+---------+---------+--------+
	// | TASK | ATTEMPT | ELAPSED | STATUS |
	// +------+---------+---------+--------+
	// +------+---------+---------+--------+
	// ❗You have 1 issue:
	//
	//  - Workflow input processing failed
	//  - Required workflow input 'HelloWorld.name' not specified
}
