package commands

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
)

type Uncolored struct{}

func NewUncolored() Uncolored {
	return Uncolored{}
}

func (w Uncolored) Primary(s string) {
	fmt.Println(s)
}

func (w Uncolored) Accent(s string) {
	fmt.Println(s)
}

func (w Uncolored) Error(s string) {
	fmt.Println(s)
}

func buildTestCommands(h, i, prompt_key string, prompt_int int) *Commands {
	cmds := New()
	cmds.CromwellClient = cromwell.New(h, i)
	cmds.Writer = NewUncolored()
	cmds.Prompt = NewForTests(prompt_key, prompt_int)
	return cmds
}

func ExampleCommands_ResourcesUsed() {
	// Read metadata mock
	content, err := ioutil.ReadFile("../../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	cmds := buildTestCommands(ts.URL, "", "", 0)
	err = cmds.ResourcesUsed(operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// +---------------+---------------+------------+---------+
	// |   RESOURCE    | NORMALIZED TO | PREEMPTIVE | NORMAL  |
	// +---------------+---------------+------------+---------+
	// | CPUs          | 1 hour        | 1440.00    | 720.00  |
	// | Memory (GB)   | 1 hour        | 2880.00    | 1440.00 |
	// | HDD disk (GB) | 1 month       | 20.00      | -       |
	// | SSD disk (GB) | 1 month       | 20.00      | 20.00   |
	// +---------------+---------------+------------+---------+
	// - Tasks with cache hit: 1
	// - Total time with running VMs: 2160h
}
