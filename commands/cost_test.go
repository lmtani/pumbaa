package commands

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestCost(t *testing.T) {
	file, _ := ioutil.ReadFile("./metadata.json")
	meta := MetadataResponse{}
	_ = json.Unmarshal(file, &meta)

	resp, _ := GetComputeCost(meta.Calls)
	if resp != 0.1 {
		t.Errorf("Expected %v, got %v", 0.1, resp)
	}
}
