package commands

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/lmtani/cromwell-cli/internal/prompt"
)

type PromptForTests struct {
	byKeyReturn   string
	byIndexReturn int
}

func (p PromptForTests) SelectByKey(taskOptions []string) (string, error) {
	return p.byKeyReturn, nil
}

func (p PromptForTests) SelectByIndex(t prompt.TemplateOptions, sfn func(input string, index int) bool, items interface{}) (int, error) {
	return p.byIndexReturn, nil
}

func NewForTests(byKey string, byIndex int) *PromptForTests {
	return &PromptForTests{
		byKeyReturn: byKey, byIndexReturn: byIndex,
	}
}

func ExampleCommands_Navigate() {
	// Mock http server
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "", "SayGoodbye", 1)
	err = cmds.Navigate(operation)
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

func ExampleCommands_Navigate_second() {
	// Mock http server
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "", "RunHelloWorkflows", 1)
	err = cmds.Navigate(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Workflow: HelloHere
	//
	// Command status: Done
	// echo "This simulates a task output file, processig string: scatter_1" > final.txt
	// Logs:
	// /home/taniguti/cromwell-executions/HelloWorld/d47bb332-78e6-4265-8eae-c9d7929f5a1c/call-RunHelloWorkflows/shard-1/execution/stderr
	// /home/taniguti/cromwell-executions/HelloWorld/d47bb332-78e6-4265-8eae-c9d7929f5a1c/call-RunHelloWorkflows/shard-1/execution/stdout
	//
	// 🐋 Docker image:
	// ubuntu:20.04
}
