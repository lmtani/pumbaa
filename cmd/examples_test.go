package cmd

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/lmtani/cromwell-cli/pkg/cromwell_client"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func BuildTestCommands(h, i string) *Commands {
	cmds := New()
	cmds.CromwellClient = cromwell_client.New(h, i)
	cmds.Writer = output.NewColoredWriter(os.Stdout)
	return cmds
}

func BuildTestServer(url, resp string, httpStatus int) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == url {
				w.WriteHeader(httpStatus)
				_, err := w.Write([]byte(resp))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	return ts
}

func BuildTestServerMutable(url string) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == url {
				resp := `{"id": "a-new-uuid", "status": "Done"}`
				_, err := w.Write([]byte(resp))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	return ts
}

func ExampleCommands_QueryWorkflow() {
	ts := BuildTestServer("/api/workflows/v1/query", `{"Results": [{"id":"aaa", "name": "wf", "status": "Running", "submission": "2021-03-22T13:06:42.626Z", "start": "2021-03-22T13:06:42.626Z", "end": "2021-03-22T13:06:42.626Z", "metadataarchivestatus": "archived"}], "TotalResultsCount": 1}`, http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
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

func ExampleCommands_Inputs() {
	// Read metadata mock
	content, err := os.ReadFile("../pkg/cromwell_client/mocks/metadata.json")
	if err != nil {
		fmt.Print("Could no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
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

func ExampleCommands_KillWorkflow() {
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/abort", `{"id": "aaa-bbb-ccc", "status": "aborting"}`, http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
	err := cmds.KillWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Operation=aaa-bbb-ccc, Status=aborting
}

func ExampleCommands_ResourcesUsed() {
	// Read metadata mock
	content, err := os.ReadFile("../pkg/cromwell_client/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
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
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/outputs", `{"id": "aaa-bbb-ccc", "outputs": {"output_path": "/path/to/output.txt"}}`, http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
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
	ts := BuildTestServer("/api/workflows/v1", `{"id": "a-new-uuid", "status": "Submitted"}`, http.StatusOK)
	defer ts.Close()

	wdlPath := "../examples/wf.wdl"
	inputsPath := "../examples/wf.inputs.json"

	cmds := BuildTestCommands(ts.URL, "")
	err := cmds.SubmitWorkflow(wdlPath, inputsPath, wdlPath, inputsPath)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// 🐖 Operation= a-new-uuid , Status=Submitted
}

func ExampleCommands_Wait() {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServerMutable("/api/workflows/v1/" + operation + "/status")
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
	err := cmds.Wait(operation, 1)
	if err != nil {
		log.Printf("Error: %#v", err)
	}

	// Output:
	// Time between status check = 1
	// Status=Done
}

func ExampleCommands_MetadataWorkflow() {
	// Read metadata mock
	content, err := os.ReadFile("../pkg/cromwell_client/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
	err = cmds.MetadataWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// +-----------------------------+---------+----------+---------------------+
	// |            TASK             | ATTEMPT | ELAPSED  |       STATUS        |
	// +-----------------------------+---------+----------+---------------------+
	// | RunHelloWorkflows (Scatter) | -       | 15.029s  | 2/2 Done | 0 Failed |
	// | SayGoodbye                  | 1       | 720h0m0s | Done                |
	// | SayHello                    | 1       | 720h0m0s | Done                |
	// | SayHelloCache               | 1       | 720h0m0s | Done                |
	// +-----------------------------+---------+----------+---------------------+
	// 🔧 Custom options
	// - delete_intermediate_output_files: true
	// - final_workflow_outputs_dir: gs://some-bucket/
	// - jes_gcs_root: gs://workspace-bucket
	// - read_from_cache: false
	// - use_relative_output_paths: false
}

func ExampleCommands_MetadataWorkflow_second() {
	// Read metadata mock
	content, err := os.ReadFile("../pkg/cromwell_client/mocks/metadata-failed.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
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

func MockSelectByKey(taskOptions []string) (string, error) {
	return "SayGoodbye", nil
}

func MockSelectByIndex(sfn func(input string, index int) bool, items interface{}) (int, error) {
	return 1, nil
}

func Example_navigate() {
	// Mock http server
	content, err := os.ReadFile("../pkg/cromwell_client/mocks/metadata.json")
	if err != nil {
		fmt.Print("Could no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	// Mock prompt interaction
	defaultPromptSelectKey = MockSelectByKey
	defaultPromptSelectIndex = MockSelectByIndex

	cmds := BuildTestCommands(ts.URL, "")

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
