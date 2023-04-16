package cmd

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOutputsHttpError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/workflows/v1/"+operation+"/outputs" {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`Workflow ID Not Found`))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
	err := cmds.OutputsWorkflow(operation)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}

func TestOutputsReturnError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/outputs", `improbable-situation`, http.StatusOK)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
	err := cmds.OutputsWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}
