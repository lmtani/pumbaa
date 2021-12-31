package commands

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lmtani/cromwell-cli/internal/util"
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

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.OutputsWorkflow(operation)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}

func TestOutputsReturnError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := util.BuildTestServer("/api/workflows/v1/"+operation+"/outputs", `improbable-situation`)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.OutputsWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}
