package prompt

import (
	"fmt"
	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/lmtani/cromwell-cli/pkg/output"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

type MockUi struct{}

func (m MockUi) SelectByKey(taskOptions []string) (string, error) {
	return "SayGoodbye", nil
}

func (m MockUi) SelectByIndex(t TemplateOptions, sfn func(input string, index int) bool, items interface{}) (int, error) {
	return 1, nil
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

func Example_navigate() {
	// Mock http server
	content, err := os.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Could no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	ui := &TermUi{
		CromwellClient: cromwell.New(ts.URL, ""),
		Writer:         output.NewColoredWriter(os.Stdout),
		Prompt:         MockUi{},
	}

	err = ui.Navigate(operation)
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
