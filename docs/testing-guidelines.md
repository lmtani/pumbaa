# Testing Guidelines

This document outlines testing conventions and best practices for the Pumbaa project.

## Project Structure

Tests are placed alongside the code they test, following Go conventions:

```
internal/
├── domain/workflow/
│   ├── entity.go
│   ├── entity_test.go
│   ├── preemption.go
│   ├── preemption_test.go
│   ├── monitoring.go
│   └── monitoring_test.go
├── application/workflow/
│   └── debuginfo/
│       ├── usecase.go
│       └── usecase_test.go
└── infrastructure/cromwell/
    ├── mapper.go
    └── mapper_test.go
```

## Test Categories by Layer

| Layer | Test Type | Dependencies | Purpose |
|-------|-----------|--------------|---------|
| Domain | Unit | None | Pure business logic |
| Application | Unit/Integration | Mocked infrastructure | Use case orchestration |
| Infrastructure | Integration | External systems or mocks | API clients, parsers |
| Interfaces | E2E (optional) | Full stack | CLI, TUI validation |

## Conventions

### Naming

- Test files: `*_test.go` in the same package
- Test functions: `TestFunctionName` or `TestTypeName_MethodName`
- Subtests: Use descriptive names with `t.Run("description", ...)`

### Table-Driven Tests

Prefer table-driven tests for multiple scenarios:

```go
func TestCalculatePreemptionSummary(t *testing.T) {
    tests := []struct {
        name     string
        workflow *Workflow
        want     *PreemptionSummary
    }{
        {
            name:     "empty workflow",
            workflow: &Workflow{},
            want:     &PreemptionSummary{CostEfficiency: 1.0},
        },
        {
            name:     "workflow with preemptions",
            workflow: createTestWorkflow(3, 2), // 3 tasks, 2 preempted
            want:     &PreemptionSummary{TotalPreemptions: 2},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.workflow.CalculatePreemptionSummary()
            // assertions
        })
    }
}
```

### Test Data

Place test fixtures in `test_data/`:

```
test_data/
├── metadata.json           # Cromwell workflow metadata
├── metadata_scattered.json # Scattered workflow example
├── monitoring.tsv          # Resource monitoring output
└── monitoring_empty.tsv    # Edge case: empty log
```

Load test data with relative paths:

```go
data, err := os.ReadFile("../../test_data/metadata.json")
```

### Assertions

Use standard library comparisons. For complex structs, consider:

```go
if got.TotalTasks != want.TotalTasks {
    t.Errorf("TotalTasks = %d, want %d", got.TotalTasks, want.TotalTasks)
}
```

For deep equality with large structs:

```go
import "reflect"

if !reflect.DeepEqual(got, want) {
    t.Errorf("got %+v, want %+v", got, want)
}
```

### Test Helpers

Create helper functions for common setup:

```go
// createTestCall creates a Call with the given parameters for testing.
func createTestCall(name string, attempt int, preemptible bool) Call {
    preemptStr := "false"
    if preemptible {
        preemptStr = "3" // 3 preemptible attempts
    }
    return Call{
        Name:        name,
        Attempt:     attempt,
        Preemptible: preemptStr,
        Status:      StatusSucceeded,
    }
}
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/domain/workflow/...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Coverage Goals

| Layer | Target Coverage |
|-------|-----------------|
| Domain | 80%+ (critical business logic) |
| Application | 70%+ |
| Infrastructure | 60%+ |

## Mocking

For infrastructure dependencies, create interfaces and mock implementations:

```go
// In domain/ports/workflow.go
type WorkflowRepository interface {
    GetMetadata(ctx context.Context, id string) (*workflow.Workflow, error)
}

// In tests
type mockRepository struct {
    metadata *workflow.Workflow
    err      error
}

func (m *mockRepository) GetMetadata(ctx context.Context, id string) (*workflow.Workflow, error) {
    return m.metadata, m.err
}
```

## Edge Cases to Test

1. **Empty inputs**: Empty slices, nil pointers, empty strings
2. **Boundary conditions**: Zero, one, many items
3. **Error conditions**: Invalid input, parsing failures
4. **Unicode/special characters**: In workflow names, labels
5. **Time zones**: Timestamp parsing with different formats
