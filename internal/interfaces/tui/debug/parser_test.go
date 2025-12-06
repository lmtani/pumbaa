package debug

import (
	"os"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	// Read the test metadata file
	data, err := os.ReadFile("../../../../test_data/metadata.json")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	// Parse the metadata
	wm, err := ParseMetadata(data)
	if err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Verify basic fields
	if wm.ID != "de8b03fd-ac06-45e8-b3c4-ef921ba0dd80" {
		t.Errorf("Expected ID 'de8b03fd-ac06-45e8-b3c4-ef921ba0dd80', got '%s'", wm.ID)
	}

	if wm.Name != "SingleSampleGenotyping" {
		t.Errorf("Expected name 'SingleSampleGenotyping', got '%s'", wm.Name)
	}

	if wm.Status != "Succeeded" {
		t.Errorf("Expected status 'Succeeded', got '%s'", wm.Status)
	}

	// Verify calls were parsed
	if len(wm.Calls) == 0 {
		t.Error("Expected calls to be parsed, got none")
	}

	// Verify a specific call exists
	if _, ok := wm.Calls["SingleSampleGenotyping.RunDeepVariant"]; !ok {
		t.Error("Expected 'SingleSampleGenotyping.RunDeepVariant' call to exist")
	}

	// Verify call details
	dvCalls := wm.Calls["SingleSampleGenotyping.RunDeepVariant"]
	if len(dvCalls) != 1 {
		t.Errorf("Expected 1 RunDeepVariant call, got %d", len(dvCalls))
	}

	dvCall := dvCalls[0]
	if dvCall.ExecutionStatus != "Done" {
		t.Errorf("Expected execution status 'Done', got '%s'", dvCall.ExecutionStatus)
	}

	if dvCall.Backend != "GoogleBatch" {
		t.Errorf("Expected backend 'GoogleBatch', got '%s'", dvCall.Backend)
	}

	if dvCall.CPU != "12" {
		t.Errorf("Expected CPU '12', got '%s'", dvCall.CPU)
	}

	if dvCall.ReturnCode == nil || *dvCall.ReturnCode != 0 {
		t.Error("Expected return code 0")
	}

	// Verify subworkflows
	if _, ok := wm.Calls["SingleSampleGenotyping.ExpansionHunterWorkflow"]; !ok {
		t.Error("Expected subworkflow 'SingleSampleGenotyping.ExpansionHunterWorkflow' to exist")
	}

	ehCalls := wm.Calls["SingleSampleGenotyping.ExpansionHunterWorkflow"]
	if len(ehCalls) != 1 {
		t.Errorf("Expected 1 ExpansionHunterWorkflow call, got %d", len(ehCalls))
	}

	if ehCalls[0].SubWorkflowID != "bddb1ecd-72d8-4607-ad51-7f5def9f460f" {
		t.Errorf("Expected subworkflow ID 'bddb1ecd-72d8-4607-ad51-7f5def9f460f', got '%s'", ehCalls[0].SubWorkflowID)
	}
}

func TestBuildTree(t *testing.T) {
	// Read the test metadata file
	data, err := os.ReadFile("../../../../test_data/metadata.json")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	// Parse the metadata
	wm, err := ParseMetadata(data)
	if err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Build the tree
	tree := BuildTree(wm)

	// Verify root node
	if tree.Name != "SingleSampleGenotyping" {
		t.Errorf("Expected root name 'SingleSampleGenotyping', got '%s'", tree.Name)
	}

	if tree.Type != NodeTypeWorkflow {
		t.Errorf("Expected root type NodeTypeWorkflow, got %v", tree.Type)
	}

	if tree.Status != "Succeeded" {
		t.Errorf("Expected root status 'Succeeded', got '%s'", tree.Status)
	}

	// Verify children exist
	if len(tree.Children) == 0 {
		t.Error("Expected tree to have children")
	}

	// Count different node types
	var callCount, subWorkflowCount int
	for _, child := range tree.Children {
		switch child.Type {
		case NodeTypeCall:
			callCount++
		case NodeTypeSubWorkflow:
			subWorkflowCount++
		}
	}

	if subWorkflowCount == 0 {
		t.Error("Expected at least one subworkflow node")
	}

	if callCount == 0 {
		t.Error("Expected at least one call node")
	}
}

func TestGetVisibleNodes(t *testing.T) {
	// Create a simple tree for testing
	root := &TreeNode{
		Name:     "root",
		Expanded: true,
		Children: []*TreeNode{
			{
				Name:     "child1",
				Expanded: false,
				Children: []*TreeNode{
					{Name: "grandchild1"},
					{Name: "grandchild2"},
				},
			},
			{
				Name:     "child2",
				Expanded: true,
				Children: []*TreeNode{
					{Name: "grandchild3"},
				},
			},
		},
	}

	// Get visible nodes
	visible := GetVisibleNodes(root)

	// Should show: root, child1, child2, grandchild3 (4 nodes)
	// child1's children should be hidden because child1 is not expanded
	if len(visible) != 4 {
		t.Errorf("Expected 4 visible nodes, got %d", len(visible))
	}

	expectedNames := []string{"root", "child1", "child2", "grandchild3"}
	for i, node := range visible {
		if node.Name != expectedNames[i] {
			t.Errorf("Expected node %d to be '%s', got '%s'", i, expectedNames[i], node.Name)
		}
	}
}

func TestStatusStyle(t *testing.T) {
	tests := []struct {
		status string
	}{
		{"Done"},
		{"Succeeded"},
		{"Failed"},
		{"Running"},
		{"Unknown"},
	}

	for _, tt := range tests {
		// Just ensure no panic
		_ = StatusStyle(tt.status)
		_ = StatusIcon(tt.status)
	}
}

func TestNodeTypeIcon(t *testing.T) {
	types := []NodeType{NodeTypeWorkflow, NodeTypeCall, NodeTypeSubWorkflow, NodeTypeShard}
	for _, nt := range types {
		// Just ensure no panic
		_ = NodeTypeIcon(nt)
	}
}
