package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// mockHealthChecker is a test double for ports.HealthChecker.
type mockHealthChecker struct {
	status *domain.HealthStatus
	err    error
}

func (m *mockHealthChecker) GetHealthStatus(ctx context.Context) (*domain.HealthStatus, error) {
	return m.status, m.err
}

const preflightWDL = `version 1.0

workflow Align {
    input {
        File reads
        String sample
        Int threads = 4
    }
}
`

// preflightFiles builds a provider serving the given WDL and inputs, with an
// optional GetSize behavior for path checks.
func preflightFiles(wdlSrc, inputs string, getSize func(ctx context.Context, path string) (int64, error)) *mockFileProvider {
	return &mockFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			switch path {
			case "align.wdl":
				return []byte(wdlSrc), nil
			case "inputs.json":
				return []byte(inputs), nil
			}
			return nil, errors.New("unexpected path: " + path)
		},
		getSizeFunc: getSize,
	}
}

func checkByName(t *testing.T, r *PreflightReport, name string) PreflightCheck {
	t.Helper()
	for _, c := range r.Checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("check %q not found in %+v", name, r.Checks)
	return PreflightCheck{}
}

func TestPreflightAllGreen(t *testing.T) {
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/r.fastq", "Align.sample": "NA12878"}`,
		func(ctx context.Context, path string) (int64, error) { return 42, nil })
	health := &mockHealthChecker{status: &domain.HealthStatus{OK: true}}
	uc := NewPreflightUseCase(fp, health)

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if report.HasErrors() {
		t.Fatalf("a valid submission must pass: %+v", report.Checks)
	}
	if report.WorkflowName != "Align" {
		t.Errorf("WorkflowName = %q, want Align", report.WorkflowName)
	}
	for _, c := range report.Checks {
		if c.Status != CheckOK {
			t.Errorf("check %q = %s (%s), want ok", c.Name, c.Status, c.Detail)
		}
	}
	errCount, warnCount := report.Counts()
	if errCount != 0 || warnCount != 0 {
		t.Errorf("counts = %d errors / %d warnings, want none", errCount, warnCount)
	}
}

func TestPreflightReportsEveryProblemAtOnce(t *testing.T) {
	// Missing required input AND a bad path: a newcomer should see both in
	// one pass, not one per attempt.
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/missing.fastq"}`,
		func(ctx context.Context, path string) (int64, error) {
			return 0, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		})
	uc := NewPreflightUseCase(fp, &mockHealthChecker{status: &domain.HealthStatus{OK: true}})

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("expected errors")
	}
	inputs := checkByName(t, report, "Inputs")
	if inputs.Status != CheckFailed {
		t.Errorf("Inputs check = %s, want failed", inputs.Status)
	}
	files := checkByName(t, report, "Input files")
	if files.Status != CheckFailed {
		t.Errorf("Input files check = %s, want failed", files.Status)
	}
	if len(files.Items) != 1 || !strings.Contains(files.Items[0].Message, "does not exist") {
		t.Errorf("missing file should be reported plainly: %+v", files.Items)
	}
	errCount, _ := report.Counts()
	if errCount != 2 {
		t.Errorf("expected both problems reported, got %d: %+v", errCount, report.Checks)
	}
}

func TestPreflightUnverifiablePathIsAWarning(t *testing.T) {
	// No local credentials must not block a submission to a Cromwell that
	// does have them.
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/r.fastq", "Align.sample": "NA12878"}`,
		func(ctx context.Context, path string) (int64, error) {
			return 0, errors.New("failed to create GCS client: no credentials found")
		})
	uc := NewPreflightUseCase(fp, &mockHealthChecker{status: &domain.HealthStatus{OK: true}})

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if report.HasErrors() {
		t.Fatalf("an unverifiable path must not block: %+v", report.Checks)
	}
	files := checkByName(t, report, "Input files")
	if files.Status != CheckWarning {
		t.Errorf("Input files check = %s, want warning", files.Status)
	}
	if len(files.Items) != 1 || !strings.Contains(files.Items[0].Message, "could not verify") {
		t.Errorf("warning should say it could not check: %+v", files.Items)
	}
}

func TestPreflightUnreachableServerFails(t *testing.T) {
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/r.fastq", "Align.sample": "NA12878"}`, nil)
	uc := NewPreflightUseCase(fp, &mockHealthChecker{err: errors.New("connection refused")})

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	server := checkByName(t, report, "Cromwell server")
	if server.Status != CheckFailed || !report.HasErrors() {
		t.Errorf("an unreachable server must fail preflight: %+v", server)
	}
	if !strings.Contains(server.Items[0].Message, "CROMWELL_HOST") {
		t.Errorf("the message should point at the fix: %q", server.Items[0].Message)
	}
}

func TestPreflightDegradedServerWarns(t *testing.T) {
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/r.fastq", "Align.sample": "NA12878"}`,
		func(ctx context.Context, path string) (int64, error) { return 1, nil })
	uc := NewPreflightUseCase(fp, &mockHealthChecker{
		status: &domain.HealthStatus{OK: false, Degraded: true, UnhealthySystems: []string{"PAPI"}},
	})

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	server := checkByName(t, report, "Cromwell server")
	if server.Status != CheckWarning || report.HasErrors() {
		t.Errorf("a degraded server should warn, not block: %+v", server)
	}
}

func TestPreflightSkipFlags(t *testing.T) {
	called := false
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/r.fastq", "Align.sample": "NA12878"}`,
		func(ctx context.Context, path string) (int64, error) {
			called = true
			return 0, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		})
	uc := NewPreflightUseCase(fp, &mockHealthChecker{err: errors.New("connection refused")})

	report, err := uc.Execute(context.Background(), PreflightInput{
		WorkflowFile: "align.wdl",
		InputsFile:   "inputs.json",
		SkipServer:   true,
		SkipPaths:    true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if called {
		t.Error("--skip-paths must not touch storage")
	}
	if report.HasErrors() {
		t.Errorf("both failing checks were skipped: %+v", report.Checks)
	}
	if s := checkByName(t, report, "Cromwell server").Status; s != CheckSkipped {
		t.Errorf("server check = %s, want skipped", s)
	}
	if s := checkByName(t, report, "Input files").Status; s != CheckSkipped {
		t.Errorf("paths check = %s, want skipped", s)
	}
}

func TestPreflightWithoutHealthCheckerSkipsServer(t *testing.T) {
	fp := preflightFiles(preflightWDL, `{"Align.reads": "gs://b/r.fastq", "Align.sample": "NA12878"}`,
		func(ctx context.Context, path string) (int64, error) { return 1, nil })
	uc := NewPreflightUseCase(fp, nil)

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if s := checkByName(t, report, "Cromwell server").Status; s != CheckSkipped {
		t.Errorf("server check = %s, want skipped when no checker is wired", s)
	}
}

func TestPreflightUnparseableWDLDoesNotBlock(t *testing.T) {
	fp := preflightFiles("this is not WDL {{{", `{"whatever": 1}`, nil)
	uc := NewPreflightUseCase(fp, nil)

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl", InputsFile: "inputs.json"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if report.HasErrors() {
		t.Errorf("Cromwell is the authority on WDL we cannot parse: %+v", report.Checks)
	}
	if s := checkByName(t, report, "WDL syntax").Status; s != CheckWarning {
		t.Errorf("WDL syntax check = %s, want warning", s)
	}
	if s := checkByName(t, report, "Inputs").Status; s != CheckSkipped {
		t.Errorf("Inputs check = %s, want skipped when the WDL is unreadable", s)
	}
}

func TestPreflightWithoutInputsFileReportsMissingInputs(t *testing.T) {
	fp := preflightFiles(preflightWDL, "", nil)
	uc := NewPreflightUseCase(fp, nil)

	report, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "align.wdl"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !report.HasErrors() {
		t.Fatal("a workflow with required inputs and no inputs file cannot run")
	}
	errCount, _ := report.Counts()
	if errCount != 2 {
		t.Errorf("expected one error per required input, got %d", errCount)
	}
}

func TestPreflightRequiresWorkflowFile(t *testing.T) {
	uc := NewPreflightUseCase(&mockFileProvider{}, nil)
	if _, err := uc.Execute(context.Background(), PreflightInput{}); err == nil {
		t.Error("expected an error without a workflow file")
	}
}

func TestPreflightUnreadableFilesAreHardErrors(t *testing.T) {
	fp := &mockFileProvider{
		readBytesFunc: func(ctx context.Context, path string) ([]byte, error) {
			return nil, errors.New("no such file")
		},
	}
	uc := NewPreflightUseCase(fp, nil)

	if _, err := uc.Execute(context.Background(), PreflightInput{WorkflowFile: "nope.wdl"}); err == nil {
		t.Error("a missing workflow file should fail outright, not as a check")
	}
}
