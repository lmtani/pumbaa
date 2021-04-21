package commands

import (
	"log"
)

func ExampleKill() {
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/abort", `{"id": "aaa-bbb-ccc", "status": "aborting"}`)
	defer ts.Close()

	err := KillWorkflow(ts.URL, "", operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// Operation=aaa-bbb-ccc, Status=aborting
}
