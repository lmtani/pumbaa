package cmd

import (
	"net/http"
	"testing"

	"github.com/lmtani/pumbaa/pkg/cromwell_client"
)

func TestInputsHttpError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", "Workflow ID Not Found", http.StatusNotFound)
	defer ts.Close()

	cromwellClient := cromwell_client.New(ts.URL, "")

	err := Inputs(operation, cromwellClient)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}
