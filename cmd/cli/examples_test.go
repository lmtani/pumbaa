package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/lmtani/pumbaa/internal/adapters/gcp"

	"github.com/lmtani/pumbaa/internal/adapters/cromwellclient"
	"github.com/lmtani/pumbaa/internal/adapters/logger"
	"github.com/lmtani/pumbaa/internal/adapters/writer"

	"github.com/lmtani/pumbaa/internal/core/cromwell"

	"github.com/lmtani/pumbaa/internal/core/interactive"
)

const (
	MetadataPath       = "../../internal/adapters/cromwellclient/testdata/metadata.json"
	MetadataFailedPath = "../../internal/adapters/cromwellclient/testdata/metadata-failed.json"
)

const (
	WDLPath    = "../../assets/workflow.wdl"
	InputsPath = "../../assets/workflow.inputs.json"
)

var AbortingResponse = `{
	"id": "aaaa-bbbb-uuid",
	"status": "aborting"
}`

var QueryResponse = `{
	"Results": [{
		"id": "aaa",
		"name": "wf",
		"status": "Running",
		"submission": "2021-03-22T13:06:42.626Z",
		"start": "2021-03-22T13:06:42.626Z",
		"end": "2021-03-22T13:06:42.626Z",
		"metadataarchivestatus": "archived"
	}],
	"TotalResultsCount": 1
}`

func NewTestHandler(h string) *Handler {
	googleClient := gcp.GCP{
		Aud:     "",
		Factory: &gcp.MockDependencyFactory{},
	}

	client := cromwellclient.NewCromwellClient(h, &googleClient)
	c := cromwell.NewCromwell(client, logger.NewLogger(logger.InfoLevel))
	return &Handler{c: c, w: writer.NewColoredWriter(os.Stdout)}
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

func Example_queryWorkflow() {
	// setup
	ts := BuildTestServer("/api/workflows/v1/query", QueryResponse, http.StatusOK)
	defer ts.Close()

	h := NewTestHandler(ts.URL)

	// call
	d, err := h.c.QueryWorkflow("wf", 0)
	if err != nil {
		log.Print(err)
	}
	h.w.QueryTable(d)
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
	h := NewTestHandler(ts.URL)

	d, err := h.c.Inputs(operation)
	if err != nil {
		log.Print(err)
	}
	err = h.w.Json(d)
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
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/abort", AbortingResponse, http.StatusOK)
	defer ts.Close()
	h := NewTestHandler(ts.URL)

	_, err := h.c.Kill(operation)
	if err != nil {
		log.Print(err)
	}
	h.w.Message(fmt.Sprintf("Operation=%s, Status=%s", operation, "aborting"))
	// Output:
	// Operation=aaaa-bbbb-uuid, Status=aborting
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
	h := NewTestHandler(ts.URL)

	d, err := h.c.ResourceUsages(operation)
	if err != nil {
		log.Print(err)
	}
	h.w.ResourceTable(d)
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
	h := NewTestHandler(ts.URL)

	d, err := h.c.Outputs(operation)
	if err != nil {
		log.Print(err)
	}
	err = h.w.Json(d.Outputs)
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
	h := NewTestHandler(ts.URL)

	d, err := h.c.SubmitWorkflow(WDLPath, InputsPath, WDLPath, InputsPath)
	if err != nil {
		log.Print(err)
	}
	h.w.Message(fmt.Sprintf("🐖 Operation= %s , Status=%s", d.ID, d.Status))
	// Output:
	// 🐖 Operation= a-new-uuid , Status=Submitted
}

func Example_wait() {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServerMutable("/api/workflows/v1/" + operation + "/status")
	defer ts.Close()
	h := NewTestHandler(ts.URL)

	err := h.c.Wait(operation, 1)
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
	h := NewTestHandler(ts.URL)

	d, err := h.c.Metadata(operation)
	if err != nil {
		log.Print(err)
	}
	err = h.w.MetadataTable(d)
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
	h := NewTestHandler(ts.URL)

	d, err := h.c.Metadata(operation)
	if err != nil {
		log.Print(err)
	}

	err = h.w.MetadataTable(d)
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
	cromwellClient := cromwellclient.NewCromwellClient(ts.URL, nil)
	w := writer.NewColoredWriter(os.Stdout)
	mockedPrompt := MockedPrompt{
		indexToReturn: 1,
		keyToReturn:   "SayGoodbye",
	}

	n := interactive.NewNavigate(cromwellClient, w, &mockedPrompt)

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
