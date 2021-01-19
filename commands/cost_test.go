package commands

import "testing"

func TestCost(t *testing.T) {
	resp := GetComputeCost("aaa")
	if resp != 0.1 {
		t.Errorf("Expected %v, got %v", 0.1, resp)
	}
}
