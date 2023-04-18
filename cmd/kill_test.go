package cmd

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lmtani/cromwell-cli/pkg/cromwell_client"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

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

	cromwellClient := cromwell_client.New(ts.URL, "")
	writer := output.NewColoredWriter(os.Stdout)

	err := KillWorkflow(operation, cromwellClient, writer)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}