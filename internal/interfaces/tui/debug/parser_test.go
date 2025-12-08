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

	// Verify subworkflow metadata is parsed
	if ehCalls[0].SubWorkflowMetadata == nil {
		t.Error("Expected subworkflow metadata to be parsed")
	} else {
		subWM := ehCalls[0].SubWorkflowMetadata
		if subWM.Name != "ExpansionHunterWorkflow" {
			t.Errorf("Expected subworkflow name 'ExpansionHunterWorkflow', got '%s'", subWM.Name)
		}
		if len(subWM.Calls) != 2 {
			t.Errorf("Expected 2 calls in subworkflow, got %d", len(subWM.Calls))
		}
		// Check that the nested calls exist
		if _, ok := subWM.Calls["ExpansionHunterWorkflow.RunExpansionHunter"]; !ok {
			t.Error("Expected 'RunExpansionHunter' call in subworkflow")
		}
		if _, ok := subWM.Calls["ExpansionHunterWorkflow.SamtoolsSort"]; !ok {
			t.Error("Expected 'SamtoolsSort' call in subworkflow")
		}
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

	// Find the ExpansionHunterWorkflow node and verify it has children
	var ehNode *TreeNode
	for _, child := range tree.Children {
		if child.Name == "ExpansionHunterWorkflow" && child.Type == NodeTypeSubWorkflow {
			ehNode = child
			break
		}
	}

	if ehNode == nil {
		t.Fatal("Expected to find ExpansionHunterWorkflow node")
	}

	// Verify subworkflow has children from its embedded metadata
	if len(ehNode.Children) != 2 {
		t.Errorf("Expected ExpansionHunterWorkflow to have 2 children, got %d", len(ehNode.Children))
	}

	// Verify child names
	childNames := make(map[string]bool)
	for _, child := range ehNode.Children {
		childNames[child.Name] = true
	}
	if !childNames["RunExpansionHunter"] {
		t.Error("Expected 'RunExpansionHunter' as child of subworkflow")
	}
	if !childNames["SamtoolsSort"] {
		t.Error("Expected 'SamtoolsSort' as child of subworkflow")
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

func TestParseMetadataWithFailures(t *testing.T) {
	// Metadata JSON with failures (workflow failed before calls)
	data := []byte(`{
		"id": "85f955f2-2cd6-4cb1-839d-2cd0f554b09b",
		"workflowName": "TestWorkflow",
		"status": "Failed",
		"calls": {},
		"outputs": {},
		"inputs": {},
		"failures": [
			{
				"message": "Workflow input processing failed",
				"causedBy": [
					{
						"message": "Required workflow input 'TestWorkflow.input1' not specified",
						"causedBy": []
					},
					{
						"message": "Required workflow input 'TestWorkflow.input2' not specified",
						"causedBy": []
					}
				]
			}
		]
	}`)

	wm, err := ParseMetadata(data)
	if err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Verify failures were parsed
	if len(wm.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(wm.Failures))
	}

	failure := wm.Failures[0]
	if failure.Message != "Workflow input processing failed" {
		t.Errorf("Expected failure message 'Workflow input processing failed', got '%s'", failure.Message)
	}

	// Verify nested causes
	if len(failure.CausedBy) != 2 {
		t.Fatalf("Expected 2 causes, got %d", len(failure.CausedBy))
	}

	expectedCauses := []string{
		"Required workflow input 'TestWorkflow.input1' not specified",
		"Required workflow input 'TestWorkflow.input2' not specified",
	}

	for i, cause := range failure.CausedBy {
		if cause.Message != expectedCauses[i] {
			t.Errorf("Expected cause %d message '%s', got '%s'", i, expectedCauses[i], cause.Message)
		}
	}
}

func TestParseMetadataWithNestedFailures(t *testing.T) {
	// Metadata JSON with deeply nested failures
	data := []byte(`{
		"id": "test-id",
		"workflowName": "TestWorkflow",
		"status": "Failed",
		"calls": {},
		"failures": [
			{
				"message": "Level 1 error",
				"causedBy": [
					{
						"message": "Level 2 error",
						"causedBy": [
							{
								"message": "Level 3 root cause",
								"causedBy": []
							}
						]
					}
				]
			}
		]
	}`)

	wm, err := ParseMetadata(data)
	if err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	if len(wm.Failures) != 1 {
		t.Fatalf("Expected 1 failure, got %d", len(wm.Failures))
	}

	// Verify nested structure
	level1 := wm.Failures[0]
	if level1.Message != "Level 1 error" {
		t.Errorf("Expected 'Level 1 error', got '%s'", level1.Message)
	}

	if len(level1.CausedBy) != 1 {
		t.Fatalf("Expected 1 cause at level 1, got %d", len(level1.CausedBy))
	}

	level2 := level1.CausedBy[0]
	if level2.Message != "Level 2 error" {
		t.Errorf("Expected 'Level 2 error', got '%s'", level2.Message)
	}

	if len(level2.CausedBy) != 1 {
		t.Fatalf("Expected 1 cause at level 2, got %d", len(level2.CausedBy))
	}

	level3 := level2.CausedBy[0]
	if level3.Message != "Level 3 root cause" {
		t.Errorf("Expected 'Level 3 root cause', got '%s'", level3.Message)
	}
}
