package cromwell

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientStatus(t *testing.T) {
	operation := "aaa-bbb-ccc"

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/workflows/v1/"+operation+"/status" {
				w.Write([]byte(`{"id": "aaa-bbb-ccc", "status": "running"}`))
			}
		}),
	)
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
	operation := "aaa-bbb-ccc"

	// Mock http server
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/workflows/v1/"+operation+"/abort" {
				w.Write([]byte(`{"id": "aaa-bbb-ccc", "status": "aborting"}`))
			}
		}),
	)
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
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/workflows/v1" {
				fmt.Println("entrou")
				w.Write([]byte(`{"id": "a-new-uuid", "status": "Submitted"}`))
			}
		}),
	)
	defer ts.Close()

	client := New(ts.URL, "")

	r := SubmitRequest{
		WorkflowSource:       "../../sample/wf.wdl",
		WorkflowInputs:       "../../sample/wf.inputs.json",
		WorkflowDependencies: "",
		WorkflowOptions:      ""}
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
