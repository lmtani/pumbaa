package commands

import (
	"log"
)

func ExampleOutputsWorkflow() {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/outputs", `{"id": "aaa-bbb-ccc", "outputs": {"output_path": "/path/to/output.txt"}}`)
	defer ts.Close()

	err := OutputsWorkflow(ts.URL, "", operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// {
	//    "output_path": "/path/to/output.txt"
	// }
}
