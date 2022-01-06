package commands_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"

	"github.com/lmtani/cromwell-cli/internal/commands"
	"github.com/lmtani/cromwell-cli/internal/util"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func BuildTestCommands(h, i, prompt_key string, prompt_int int) *commands.Commands {
	cmds := commands.New()
	cmds.CromwellClient = cromwell.New(h, i)
	cmds.Writer = output.NewColoredWriter(os.Stdout)
	cmds.Prompt = util.NewForTests(prompt_key, prompt_int)
	return cmds
}

func Example_cmds_QueryWorkflow() {
	ts := util.BuildTestServer("/api/workflows/v1/query", `{"Results": [{"id":"aaa", "name": "wf", "status": "Running", "submission": "2021-03-22T13:06:42.626Z", "start": "2021-03-22T13:06:42.626Z", "end": "2021-03-22T13:06:42.626Z", "metadataarchivestatus": "archived"}], "TotalResultsCount": 1}`)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.QueryWorkflow("wf", 0)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// +-----------+------+-------------------+----------+---------+
	// | OPERATION | NAME |       START       | DURATION | STATUS  |
	// +-----------+------+-------------------+----------+---------+
	// | aaa       | wf   | 2021-03-22 13h06m | 0s       | Running |
	// +-----------+------+-------------------+----------+---------+
	// - Found 1 workflows
}

func Example_cmds_Inputs() {
	// Read metadata mock
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err = cmds.Inputs(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// {
	//    "HelloHere.someFile": "gs://just-testing/file.txt",
	//    "HelloHere.someInput": "just testing string"
	// }
}

func Example_cmds_Kill() {
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/abort", `{"id": "aaa-bbb-ccc", "status": "aborting"}`)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.KillWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Operation=aaa-bbb-ccc, Status=aborting
}

func Example_cmds_Navigate() {
	// Mock http server
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "SayGoodbye", 1)
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

func Example_cmds_Navigate_second() {
	// Mock http server
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "RunHelloWorkflows", 1)
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

func Example_cmds_ResourcesUsed() {
	// Read metadata mock
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err = cmds.ResourcesUsed(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// +---------------+---------------+------------+---------+
	// |   RESOURCE    | NORMALIZED TO | PREEMPTIVE | NORMAL  |
	// +---------------+---------------+------------+---------+
	// | CPUs          | 1 hour        | 1440.00    | 720.00  |
	// | Memory (GB)   | 1 hour        | 2880.00    | 1440.00 |
	// | HDD disk (GB) | 1 month       | 20.00      | -       |
	// | SSD disk (GB) | 1 month       | 20.00      | 20.00   |
	// +---------------+---------------+------------+---------+
	// - Tasks with cache hit: 1
	// - Total time with running VMs: 2160h
}

func ExampleCommands_OutputsWorkflow() {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/outputs", `{"id": "aaa-bbb-ccc", "outputs": {"output_path": "/path/to/output.txt"}}`)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.OutputsWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// {
	//    "output_path": "/path/to/output.txt"
	// }
}

func ExampleCommands_SubmitWorkflow() {
	// Mock http server
	ts := util.BuildTestServer("/api/workflows/v1", `{"id": "a-new-uuid", "status": "Submitted"}`)
	defer ts.Close()

	wdlPath := "../../sample/wf.wdl"
	inputsPath := "../../sample/wf.inputs.json"

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.SubmitWorkflow(wdlPath, inputsPath, wdlPath, inputsPath)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// 🐖 Operation= a-new-uuid , Status=Submitted
}

func ExampleCommands_Wait() {
	// Mock http server
	rand.Seed(3)
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServerMutable("/api/workflows/v1/" + operation + "/status")
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.Wait(operation, 1, false)
	if err != nil {
		log.Printf("Error: %#v", err)
	}

	// Output:
	// Status=Done
	// Time between status check = 1
}
