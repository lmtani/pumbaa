package commands

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func ExampleKill() {
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/abort", `{"id": "aaa-bbb-ccc", "status": "aborting"}`)
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "")
	err := cmds.KillWorkflow(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Operation=aaa-bbb-ccc, Status=aborting
}

func TestKillHttpError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/workflows/v1/"+operation+"/abort" {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`Workflow ID Not Found`))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "")
	err := cmds.KillWorkflow(operation)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}
