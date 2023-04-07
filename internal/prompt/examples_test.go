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

func BuildTestPrompt(h, i string) *Ui {
	ui := New()
	ui.CromwellClient = cromwell.New(h, i)
	ui.Writer = output.NewColoredWriter(os.Stdout)
	return ui
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
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	//cmds := BuildTestPrompt(ts.URL, "", "SayGoodbye", 1)
	cmds := BuildTestPrompt(ts.URL, "")
	err = cmds.Navigate(operation)
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

func Example_navigate_second() {
	// Mock http server
	content, err := os.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := BuildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content), http.StatusOK)
	defer ts.Close()

	//cmds := BuildTestCommands(ts.URL, "", "RunHelloWorkflows", 1)
	cmds := BuildTestPrompt(ts.URL, "")
	err = cmds.Navigate(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Workflow: HelloHere
	//
	// Command status: Done
	// echo "This simulates a task output file, processig string: scatter_1" > final.txt
	// Logs:
	// /home/taniguti/cromwell-executions/HelloWorld/d47bb332-78e6-4265-8eae-c9d7929f5a1c/call-RunHelloWorkflows/shard-1/execution/stderr
	// /home/taniguti/cromwell-executions/HelloWorld/d47bb332-78e6-4265-8eae-c9d7929f5a1c/call-RunHelloWorkflows/shard-1/execution/stdout
	//
	// 🐋 Docker image:
	// ubuntu:20.04
}
