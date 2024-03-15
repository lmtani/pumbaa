package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/core"
)

const (
	MetadataPath       = "../../internal/adapters/testdata/metadata.json"
	MetadataFailedPath = "../../internal/adapters/testdata/metadata-failed.json"
)

const (
	WDLPath    = "../../assets/workflow.wdl"
	InputsPath = "../../assets/workflow.inputs.json"
)

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

func Example_queryWorkflow() {
	// setup
	ts := BuildTestServer("/api/workflows/v1/query", `{"Results": [{"id":"aaa", "name": "wf", "status": "Running", "submission": "2021-03-22T13:06:42.626Z", "start": "2021-03-22T13:06:42.626Z", "end": "2021-03-22T13:06:42.626Z", "metadataarchivestatus": "archived"}], "TotalResultsCount": 1}`, http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	q := core.NewQuery(cromwellClient, writer)

	// call
	err := q.QueryWorkflow("wf", 0)
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

func Example_inputs() {
	// Read metadata mock
	content, err := os.ReadFile(MetadataPath)
	if err != nil {
		fmt.Print("Could no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	i := core.NewInputs(cromwellClient)

	_, err = i.Inputs(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// {
	//    "HelloHere.someFile": "gs://just-testing/file.txt",
	//    "HelloHere.someInput": "just testing string"
	// }
}

func Example_killWorkflow() {
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/abort", `{"id": "aaa-bbb-ccc", "status": "aborting"}`, http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	w := adapters.NewColoredWriter(os.Stdout)
	k := core.NewKill(cromwellClient, w)

	_, err := k.Kill(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Operation=aaa-bbb-ccc, Status=aborting
}

func Example_resourcesUsed() {
	// Read metadata mock
	content, err := os.ReadFile(MetadataPath)
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	r := core.NewResourcesUsed(cromwellClient, writer)

	err = r.Get(operation)
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

func Example_outputsWorkflow() {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/outputs", `{"id": "aaa-bbb-ccc", "outputs": {"output_path": "/path/to/output.txt"}}`, http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	o := core.NewOutputs(cromwellClient)

	err := o.Outputs(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// {
	//    "output_path": "/path/to/output.txt"
	// }
}

func Example_submitWorkflow() {
	// Mock http server
	ts := BuildTestServer("/api/workflows/v1", `{"id": "a-new-uuid", "status": "Submitted"}`, http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	s := core.NewSubmit(cromwellClient, writer)

	err := s.SubmitWorkflow(WDLPath, InputsPath, WDLPath, InputsPath)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// 🐖 Operation= a-new-uuid , Status=Submitted
}

func Example_wait() {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServerMutable("/api/workflows/v1/" + operation + "/status")
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	w := core.NewWait(cromwellClient, writer)

	err := w.Wait(operation, 1)
	if err != nil {
		log.Printf("Error: %#v", err)
	}

	// Output:
	// Time between status check = 1
	// Status=Done
}

func Example_metadataWorkflow() {
	// Read metadata mock
	content, err := os.ReadFile(MetadataPath)
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	m := core.NewMetadata(cromwellClient, writer)

	err = m.Metadata(operation)
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

func Example_metadataWorkflow_second() {
	// Read metadata mock
	content, err := os.ReadFile(MetadataFailedPath)
	if err != nil {
		fmt.Printf("Coult no read metadata mock file %v", MetadataFailedPath)
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	m := core.NewMetadata(cromwellClient, writer)

	err = m.Metadata(operation)
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

type MockedPrompt struct {
	keyToReturn   string
	indexToReturn int
}

func (m *MockedPrompt) SelectByKey(taskOptions []string) (string, error) {
	return m.keyToReturn, nil
}

func (m *MockedPrompt) SelectByIndex(sfn func(input string, index int) bool, items interface{}) (int, error) {
	return m.indexToReturn, nil
}

func Example_navigate() {
	// Mock http server
	content, err := os.ReadFile(MetadataPath)
	if err != nil {
		fmt.Printf("Could no read metadata mock file %s", MetadataPath)
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	cromwellClient := DefaultCromwellClient(ts.URL, "")
	writer := adapters.NewColoredWriter(os.Stdout)
	mockedPrompt := MockedPrompt{
		indexToReturn: 1,
		keyToReturn:   "SayGoodbye",
	}
	n := core.NewNavigate(cromwellClient, writer, &mockedPrompt)

	err = n.Navigate(operation)
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
