package commands

import (
	"fmt"
	"io/ioutil"
	"log"
)

func ExampleInputs() {
	// Read metadata mock
	content, err := ioutil.ReadFile("../pkg/cromwell/mocks/metadata.json")
	if err != nil {
		fmt.Print("Coult no read metadata mock file metadata.json")
	}

	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/metadata", string(content))
	defer ts.Close()

	err = Inputs(ts.URL, "", operation)
	if err != nil {
		log.Print(err)
	}
	// Output:
	// {
	//    "HelloHere.someFile": "gs://just-testing/file.txt",
	//    "HelloHere.someInput": "just testing string"
	// }
}
