package cromwell

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

const operation string = "aaa-bbb-ccc"

func buildTestServer(url, resp string) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == url {
				_, err := w.Write([]byte(resp))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	return ts
}

func TestClientStatus(t *testing.T) {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1/"+operation+"/status", `{"id": "aaa-bbb-ccc", "status": "running"}`)

	defer ts.Close()

	client := New(ts.URL, "")

	resp, _ := client.Status(operation)

	if resp.Status != "running" {
		t.Errorf("Expected %v, got %v", "running", resp.Status)
	}
	if resp.ID != operation {
		t.Errorf("Expected %v, got %v", operation, resp.Status)
	}
}

func TestClientKill(t *testing.T) {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1/"+operation+"/abort", `{"id": "aaa-bbb-ccc", "status": "aborting"}`)
	defer ts.Close()

	client := New(ts.URL, "")

	resp, _ := client.Kill(operation)

	expected := "aborting"
	if resp.Status != expected {
		t.Errorf("Expected %v, got %v", expected, resp.Status)
	}
	if resp.ID != operation {
		t.Errorf("Expected %v, got %v", operation, resp.Status)
	}
}

func TestClientSubmit(t *testing.T) {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1", `{"id": "a-new-uuid", "status": "Submitted"}`)
	defer ts.Close()

	client := New(ts.URL, "")

	r := SubmitRequest{
		WorkflowSource:       "../../sample/wf.wdl",
		WorkflowInputs:       "../../sample/wf.inputs.json",
		WorkflowDependencies: "../../sample/wf.wdl",
		WorkflowOptions:      "../../sample/wf.inputs.json"}
	resp, _ := client.Submit(r)

	expected := "Submitted"
	if resp.Status != expected {
		t.Errorf("Expected %v, got %v", expected, resp.Status)
	}
	expectedUUID := "a-new-uuid"
	if resp.ID != expectedUUID {
		t.Errorf("Expected %v, got %v", expectedUUID, resp.Status)
	}
}

func TestClientOutputs(t *testing.T) {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1/"+operation+"/outputs", `{"id": "aaa-bbb-ccc", "outputs": {"output_path": "/path/to/output.txt"}}`)
	defer ts.Close()

	client := New(ts.URL, "")

	resp, _ := client.Outputs(operation)

	if resp.ID != operation {
		t.Errorf("Expected %v, got %v", operation, resp.ID)
	}
	outputs := map[string]interface{}{"output_path": "/path/to/output.txt"}

	if resp.Outputs["output_path"] != outputs["output_path"] {
		t.Errorf("Expected %v, got %v", outputs, resp.Outputs)
	}
}

func TestClientQuery(t *testing.T) {
	// Mock http server
	ts := buildTestServer("/api/workflows/v1/query", `{"Results": [{"id":"aaa", "name": "wf", "status": "Running", "submission": "2021-03-22T13:06:42.626Z", "start": "2021-03-22T13:06:42.626Z", "end": "2021-03-22T13:06:42.626Z", "metadataarchivestatus": "archived"}], "TotalResultsCount": 1}`)
	defer ts.Close()

	client := New(ts.URL, "")

	resp, _ := client.Query(ParamsQueryGet{})

	expectedCound := 1
	if resp.TotalResultsCount != expectedCound {
		t.Errorf("Expected %v, got %v", expectedCound, resp.TotalResultsCount)
	}

	totalWorkflows := len(resp.Results)
	if totalWorkflows != 1 {
		t.Errorf("Expected %v, got %v", 1, totalWorkflows)
	}
}

func TestClientMetadata(t *testing.T) {
	// Read metadata mock
	content, err := ioutil.ReadFile("mocks/metadata.json")
	if err != nil {
		t.Error("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	client := New(ts.URL, "")

	resp, _ := client.Metadata(operation, ParamsMetadataGet{})

	expectedWorkflowName := "HelloHere"
	if resp.WorkflowName != expectedWorkflowName {
		t.Errorf("Expected %v, got %v", expectedWorkflowName, resp.WorkflowName)
	}

	totalCalls := len(resp.Calls)
	if totalCalls != 5 {
		t.Errorf("Expected %v, got %v", 5, totalCalls)
	}

	subworkflowName := resp.Calls["HelloHere.ScatterSubworkflow"][0].SubWorkflowMetadata.WorkflowName
	expected := "ScatterSubworkflow"
	if subworkflowName != expected {
		t.Errorf("Expected %v, got %v", expected, subworkflowName)
	}
}

func TestSetupClient(t *testing.T) {
	c := Default()
	expectedHost := "http://127.0.0.1:8000"
	if c.Host != expectedHost {
		t.Errorf("Expected %v, got %v", expectedHost, c.Host)
	}
	if c.Iap != "" {
		t.Errorf("Expected %v, got %v", "", c.Iap)
	}

	newHost := "http://my-cromwell.com"
	iapCred := "iap-credential@foo"
	c.Setup(newHost, iapCred)

	if c.Host != newHost {
		t.Errorf("Expected %v, got %v", newHost, c.Host)
	}
	if c.Iap != iapCred {
		t.Errorf("Expected %v, got %v", iapCred, c.Iap)
	}

}
