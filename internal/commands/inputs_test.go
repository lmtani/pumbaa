package commands

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lmtani/cromwell-cli/internal/util"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
)

func BuildTestCommands(h, i, prompt_key string, prompt_int int) *Commands {
	cmds := New()
	cmds.CromwellClient = cromwell.New(h, i)
	cmds.Writer = output.NewColoredWriter(os.Stdout)
	cmds.Prompt = util.NewForTests(prompt_key, prompt_int)
	return cmds
}

func TestInputsHttpError(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/workflows/v1/"+operation+"/metadata" {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`Workflow ID Not Found`))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	defer ts.Close()

	cmds := BuildTestCommands(ts.URL, "", "", 0)
	err := cmds.Inputs(operation)
	if err == nil {
		t.Error("Not found error expected, nil returned")
	}
}
