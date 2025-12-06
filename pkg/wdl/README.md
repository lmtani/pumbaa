# WDL Parser Package

A Go package for parsing WDL (Workflow Description Language) files, analyzing dependencies, and creating self-contained bundles.

## Features

- **Parse WDL 1.0/1.1 files** into an Abstract Syntax Tree (AST)
- **Analyze dependencies** - resolve direct and transitive imports
- **Create bundles** - ZIP archives with all required WDL files
- **Detect circular dependencies**
- **Support for all WDL constructs** - workflows, tasks, structs, types

## Installation

```go
import "github.com/lmtani/pumbaa/pkg/wdl"
```

## Usage

### Parsing a WDL file

```go
// Parse from file
doc, err := wdl.Parse("workflow.wdl")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("WDL Version: %s\n", doc.Version)
fmt.Printf("Workflow: %s\n", doc.Workflow.Name)
fmt.Printf("Imports: %d\n", len(doc.Imports))
fmt.Printf("Tasks: %d\n", len(doc.Tasks))

// Parse from bytes
content := []byte(`version 1.0

workflow Hello {
    output {
        String msg = "Hello, World!"
    }
}
`)
doc, err := wdl.ParseBytes(content)
```

### Analyzing Dependencies

```go
// Analyze dependencies from file
graph, err := wdl.AnalyzeDependenciesFromFile("workflow.wdl")
if err != nil {
    log.Fatal(err)
}

// Get all dependencies (including transitive)
allDeps := graph.GetAllDependencies()
fmt.Printf("Total dependencies: %d\n", len(allDeps))

// Get direct dependencies only
directDeps := graph.GetDirectDependencies()
fmt.Printf("Direct dependencies: %d\n", len(directDeps))

// Print dependency graph
fmt.Println(graph.PrintGraph())
```

### Creating a Bundle

```go
// Create a bundle with all dependencies
err := wdl.CreateBundle("workflow.wdl", "bundle.zip")
if err != nil {
    log.Fatal(err)
}

// Create bundle with custom options
opts := wdl.BundleOptions{
    IncludeMetadata:            true,
    PreserveDirectoryStructure: true,
}
err = wdl.CreateBundleWithOptions("workflow.wdl", "bundle.zip", opts)

// Build bundle in memory
bundle, err := wdl.BuildBundle("workflow.wdl", wdl.DefaultBundleOptions())
if err != nil {
    log.Fatal(err)
}

// List files in bundle
for _, file := range bundle.ListFiles() {
    fmt.Println(file)
}

// Get file content
content, ok := bundle.GetFile("tasks/helper.wdl")
if ok {
    fmt.Println(string(content))
}
```

### Extracting a Bundle

```go
err := wdl.ExtractBundle("bundle.zip", "output_dir")
if err != nil {
    log.Fatal(err)
}
```

## AST Structure

The package provides detailed AST structures for WDL elements:

- `Document` - Root of the AST, contains version, imports, tasks, workflow
- `Workflow` - Workflow definition with inputs, outputs, calls, scatters, conditionals
- `Task` - Task definition with inputs, outputs, command, runtime
- `Import` - Import statement with URI and aliases
- `Call` - Call to a task or subworkflow
- `Scatter` - Scatter block
- `Conditional` - If block
- `Declaration` - Variable declaration with type and optional expression
- `Type` - WDL type (String, Int, File, Array, Map, Pair, etc.)

## Supported WDL Features

- [x] WDL 1.0 and 1.1 syntax
- [x] Workflows and tasks
- [x] Structs
- [x] All primitive types (String, Int, Float, Boolean, File)
- [x] Compound types (Array, Map, Pair)
- [x] Optional types
- [x] Imports with aliases
- [x] Calls with inputs
- [x] Scatter blocks
- [x] Conditional (if) blocks
- [x] Runtime sections
- [x] Meta and parameter_meta sections

## Dependencies

This package uses ANTLR4 runtime for parsing:
- `github.com/antlr4-go/antlr/v4`

The parser is generated from the official [OpenWDL grammar](https://github.com/openwdl/wdl).

## Regenerating the Parser

To regenerate the parser from the grammar files:

```bash
cd pkg/wdl/parser
java -jar antlr-4.13.1-complete.jar -Dlanguage=Go -package parser -visitor -listener WdlV1_1Lexer.g4 WdlV1_1Parser.g4
```

## License

MIT
