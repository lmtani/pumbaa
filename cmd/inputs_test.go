package cmd

import (
	"net/http"
	"testing"
)

func TestInputsHttpError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", "Workflow ID Not Found", http.StatusNotFound)
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "")
	err := cmds.Inputs(operation)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}
