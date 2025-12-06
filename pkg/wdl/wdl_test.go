package wdl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantVersion string
		wantImports int
		wantTasks   int
		wantWfName  string
		wantErr     bool
	}{
		{
			name: "simple workflow",
			content: `version 1.0

workflow HelloWorld {
    input {
        String name
    }
    output {
        String greeting = "Hello, " + name
    }
}
`,
			wantVersion: "1.0",
			wantImports: 0,
			wantTasks:   0,
			wantWfName:  "HelloWorld",
			wantErr:     false,
		},
		{
			name: "workflow with task",
			content: `version 1.0

task SayHello {
    input {
        String name
    }
    command {
        echo "Hello ~{name}!"
    }
    output {
        String greeting = read_string(stdout())
    }
}

workflow HelloWorkflow {
    input {
        String name
    }
    call SayHello {
        input:
            name = name
    }
    output {
        String out = SayHello.greeting
    }
}
`,
			wantVersion: "1.0",
			wantImports: 0,
			wantTasks:   1,
			wantWfName:  "HelloWorkflow",
			wantErr:     false,
		},
		{
			name: "workflow with imports",
			content: `version 1.0

import "tasks/module.wdl" as module
import "other.wdl"

workflow Main {
    call module.Task1 {}
}
`,
			wantVersion: "1.0",
			wantImports: 2,
			wantTasks:   0,
			wantWfName:  "Main",
			wantErr:     false,
		},
		{
			name: "invalid syntax",
			content: `version 1.0

workflow Invalid {
    invalid syntax here
}
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseBytes([]byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if doc.Version != tt.wantVersion {
				t.Errorf("version = %q, want %q", doc.Version, tt.wantVersion)
			}

			if len(doc.Imports) != tt.wantImports {
				t.Errorf("imports count = %d, want %d", len(doc.Imports), tt.wantImports)
			}

			if len(doc.Tasks) != tt.wantTasks {
				t.Errorf("tasks count = %d, want %d", len(doc.Tasks), tt.wantTasks)
			}

			if doc.Workflow == nil {
				t.Fatal("expected workflow, got nil")
			}
			if doc.Workflow.Name != tt.wantWfName {
				t.Errorf("workflow name = %q, want %q", doc.Workflow.Name, tt.wantWfName)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	// Test with existing test data
	testFile := "../../test_data/wdl/hello.wdl"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test file not found, skipping")
	}

	doc, err := Parse(testFile)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	if doc.Version != "1.0" {
		t.Errorf("version = %q, want %q", doc.Version, "1.0")
	}

	if len(doc.Imports) != 2 {
		t.Errorf("imports count = %d, want %d", len(doc.Imports), 2)
	}

	if doc.Workflow == nil || doc.Workflow.Name != "Hello" {
		t.Errorf("expected workflow named 'Hello'")
	}
}

func TestParseTypes(t *testing.T) {
	content := `version 1.0

task TypeTest {
    input {
        String s
        Int i
        Float f
        Boolean b
        File file
        Array[String] arr
        Array[File]+ nonEmptyArr
        Map[String, Int] map
        Pair[String, Int] pair
        String? optional
        Array[File?] optionalElements
    }
    command {
        echo "test"
    }
    output {
        String out = "test"
    }
}
`

	doc, err := ParseBytes([]byte(content))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(doc.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(doc.Tasks))
	}

	task := doc.Tasks[0]
	if len(task.Inputs) != 11 {
		t.Errorf("expected 11 inputs, got %d", len(task.Inputs))
	}

	// Check specific types
	typeTests := []struct {
		name     string
		typeStr  string
		optional bool
	}{
		{"s", "String", false},
		{"i", "Int", false},
		{"f", "Float", false},
		{"b", "Boolean", false},
		{"file", "File", false},
		{"arr", "Array[String]", false},
		{"nonEmptyArr", "Array[File]+", false},
		{"map", "Map[String, Int]", false},
		{"pair", "Pair[String, Int]", false},
		{"optional", "String?", true},
	}

	inputMap := make(map[string]string)
	for _, inp := range task.Inputs {
		if inp.Type != nil {
			inputMap[inp.Name] = inp.Type.String()
		}
	}

	for _, tt := range typeTests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := inputMap[tt.name]; !ok || got != tt.typeStr {
				t.Errorf("input %s type = %q, want %q", tt.name, got, tt.typeStr)
			}
		})
	}
}

func TestParseWorkflowWithCalls(t *testing.T) {
	content := `version 1.0

task Task1 {
    command { echo "1" }
    output { String out = read_string(stdout()) }
}

workflow TestCalls {
    input {
        String name
    }

    call Task1 {
        input:
            name = name
    }

    call Task1 as AliasedCall {
        input:
            name = "aliased"
    }

    scatter (i in [1, 2, 3]) {
        call Task1 as ScatteredTask {
            input:
                name = i
        }
    }

    if (name == "conditional") {
        call Task1 as ConditionalTask {}
    }

    output {
        String result = Task1.out
    }
}
`

	doc, err := ParseBytes([]byte(content))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	wf := doc.Workflow
	if wf == nil {
		t.Fatal("expected workflow")
	}

	if len(wf.Calls) != 2 {
		t.Errorf("expected 2 direct calls, got %d", len(wf.Calls))
	}

	if len(wf.Scatters) != 1 {
		t.Errorf("expected 1 scatter, got %d", len(wf.Scatters))
	}

	if len(wf.Conditionals) != 1 {
		t.Errorf("expected 1 conditional, got %d", len(wf.Conditionals))
	}

	// Check aliased call
	foundAlias := false
	for _, call := range wf.Calls {
		if call.Alias == "AliasedCall" {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Error("expected to find aliased call 'AliasedCall'")
	}
}

func TestParseStruct(t *testing.T) {
	content := `version 1.0

struct Person {
    String name
    Int age
    String? email
}

workflow UseStruct {
    input {
        Person p
    }
    output {
        String name = p.name
    }
}
`

	doc, err := ParseBytes([]byte(content))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(doc.Structs) != 1 {
		t.Fatalf("expected 1 struct, got %d", len(doc.Structs))
	}

	s := doc.Structs[0]
	if s.Name != "Person" {
		t.Errorf("struct name = %q, want %q", s.Name, "Person")
	}

	if len(s.Members) != 3 {
		t.Errorf("expected 3 members, got %d", len(s.Members))
	}
}

func TestParseImportAliases(t *testing.T) {
	content := `version 1.0

import "module.wdl" as mod alias Task1 as T1 alias Task2 as T2

workflow Main {}
`

	doc, err := ParseBytes([]byte(content))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(doc.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(doc.Imports))
	}

	imp := doc.Imports[0]
	if imp.URI != "module.wdl" {
		t.Errorf("import URI = %q, want %q", imp.URI, "module.wdl")
	}

	if imp.As != "mod" {
		t.Errorf("import alias = %q, want %q", imp.As, "mod")
	}

	if len(imp.Aliases) != 2 {
		t.Errorf("expected 2 member aliases, got %d", len(imp.Aliases))
	}
}

func TestParseTaskWithMeta(t *testing.T) {
	content := `version 1.0

task MetaTask {
    meta {
        description: "A test task"
        author: "Test"
        tags: ["test", "example"]
    }

    parameter_meta {
        input1: {
            description: "First input",
            required: true
        }
    }

    input {
        String input1
    }

    command {
        echo ~{input1}
    }

    output {
        String out = read_string(stdout())
    }

    runtime {
        docker: "ubuntu:latest"
        memory: "2G"
    }
}
`

	doc, err := ParseBytes([]byte(content))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(doc.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(doc.Tasks))
	}

	task := doc.Tasks[0]

	// Check meta
	if len(task.Meta) == 0 {
		t.Error("expected meta section")
	}

	// Check runtime
	if len(task.Runtime) == 0 {
		t.Error("expected runtime section")
	}
}

func TestAnalyzeDependencies(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "wdl-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	mainWDL := `version 1.0

import "tasks/task1.wdl"
import "tasks/task2.wdl"

workflow Main {
    call task1.DoSomething {}
    call task2.DoOther {}
}
`
	task1WDL := `version 1.0

task DoSomething {
    command { echo "1" }
    output { String out = read_string(stdout()) }
}
`
	task2WDL := `version 1.0

import "../common/utils.wdl"

task DoOther {
    command { echo "2" }
    output { String out = read_string(stdout()) }
}
`
	utilsWDL := `version 1.0

task Helper {
    command { echo "utils" }
    output { String out = read_string(stdout()) }
}
`

	// Create directory structure
	tasksDir := filepath.Join(tmpDir, "tasks")
	commonDir := filepath.Join(tmpDir, "common")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(commonDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write files
	if err := os.WriteFile(filepath.Join(tmpDir, "main.wdl"), []byte(mainWDL), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "task1.wdl"), []byte(task1WDL), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "task2.wdl"), []byte(task2WDL), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(commonDir, "utils.wdl"), []byte(utilsWDL), 0644); err != nil {
		t.Fatal(err)
	}

	// Analyze dependencies
	mainPath := filepath.Join(tmpDir, "main.wdl")
	graph, err := AnalyzeDependenciesFromFile(mainPath)
	if err != nil {
		t.Fatalf("failed to analyze dependencies: %v", err)
	}

	// Check results
	if graph.Root != mainPath {
		t.Errorf("root = %q, want %q", graph.Root, mainPath)
	}

	allDeps := graph.GetAllDependencies()
	if len(allDeps) != 3 {
		t.Errorf("expected 3 total dependencies, got %d", len(allDeps))
	}

	directDeps := graph.GetDirectDependencies()
	if len(directDeps) != 2 {
		t.Errorf("expected 2 direct dependencies, got %d", len(directDeps))
	}
}

func TestAnalyzeDependenciesCircular(t *testing.T) {
	// Create temp directory with circular imports
	tmpDir, err := os.MkdirTemp("", "wdl-circular-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create circular dependency
	fileA := `version 1.0
import "b.wdl"
workflow A {}
`
	fileB := `version 1.0
import "a.wdl"
workflow B {}
`

	if err := os.WriteFile(filepath.Join(tmpDir, "a.wdl"), []byte(fileA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.wdl"), []byte(fileB), 0644); err != nil {
		t.Fatal(err)
	}

	// Should detect circular dependency
	_, err = AnalyzeDependenciesFromFile(filepath.Join(tmpDir, "a.wdl"))
	if err == nil {
		t.Error("expected error for circular dependency, got nil")
	}
}

func TestCreateBundle(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "wdl-bundle-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	mainWDL := `version 1.0

import "tasks/helper.wdl" as helper

workflow Main {
    call helper.Help {}
    output {
        String out = Help.msg
    }
}
`
	helperWDL := `version 1.0

task Help {
    command { echo "help" }
    output { String msg = read_string(stdout()) }
}
`

	tasksDir := filepath.Join(tmpDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "main.wdl"), []byte(mainWDL), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tasksDir, "helper.wdl"), []byte(helperWDL), 0644); err != nil {
		t.Fatal(err)
	}

	// Create bundle
	bundlePath := filepath.Join(tmpDir, "bundle.zip")
	err = CreateBundle(filepath.Join(tmpDir, "main.wdl"), bundlePath)
	if err != nil {
		t.Fatalf("failed to create bundle: %v", err)
	}

	// Verify bundle exists
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		t.Error("bundle file was not created")
	}

	// Extract and verify
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := ExtractBundle(bundlePath, extractDir); err != nil {
		t.Fatalf("failed to extract bundle: %v", err)
	}

	// Check files exist
	if _, err := os.Stat(filepath.Join(extractDir, "main.wdl")); os.IsNotExist(err) {
		t.Error("main.wdl not found in extracted bundle")
	}
	if _, err := os.Stat(filepath.Join(extractDir, "tasks", "helper.wdl")); os.IsNotExist(err) {
		t.Error("tasks/helper.wdl not found in extracted bundle")
	}
	if _, err := os.Stat(filepath.Join(extractDir, "manifest.json")); os.IsNotExist(err) {
		t.Error("manifest.json not found in extracted bundle")
	}
}

func TestBuildBundle(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "wdl-build-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainWDL := `version 1.0

workflow Simple {
    output {
        String msg = "hello"
    }
}
`

	if err := os.WriteFile(filepath.Join(tmpDir, "simple.wdl"), []byte(mainWDL), 0644); err != nil {
		t.Fatal(err)
	}

	bundle, err := BuildBundle(filepath.Join(tmpDir, "simple.wdl"), DefaultBundleOptions())
	if err != nil {
		t.Fatalf("failed to build bundle: %v", err)
	}

	// Check bundle content
	if len(bundle.Files) != 1 {
		t.Errorf("expected 1 file in bundle, got %d", len(bundle.Files))
	}

	content, err := bundle.GetMainWorkflowContent()
	if err != nil {
		t.Fatalf("failed to get main workflow content: %v", err)
	}

	if string(content) != mainWDL {
		t.Error("main workflow content mismatch")
	}

	// Check metadata
	if bundle.Metadata == nil {
		t.Fatal("expected metadata")
	}

	if bundle.Metadata.WDLVersion != "1.0" {
		t.Errorf("WDL version = %q, want %q", bundle.Metadata.WDLVersion, "1.0")
	}

	if bundle.Metadata.MainWorkflow != "simple.wdl" {
		t.Errorf("main workflow = %q, want %q", bundle.Metadata.MainWorkflow, "simple.wdl")
	}
}

func TestParseExistingTestData(t *testing.T) {
	// Test with existing test data in the repository
	testFile := "../../test_data/wdl/hello.wdl"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test file not found, skipping")
	}

	graph, err := AnalyzeDependenciesFromFile(testFile)
	if err != nil {
		t.Fatalf("failed to analyze dependencies: %v", err)
	}

	// The hello.wdl imports two files
	deps := graph.GetAllDependencies()
	if len(deps) < 2 {
		t.Errorf("expected at least 2 dependencies, got %d", len(deps))
	}

	t.Logf("Dependency graph:\n%s", graph.PrintGraph())
}
