// preflight.go answers "is this submission going to work?" before time and
// money are spent: server reachable, WDL parseable, inputs complete and
// plausible, and the file paths they point at actually there.
package workflow

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/lmtani/pumbaa/internal/application"
	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// pathCheckConcurrency bounds the parallel existence checks: enough to keep
// a scattered input snappy, small enough to stay polite to the storage API.
const pathCheckConcurrency = 8

// CheckStatus is the outcome of one preflight check.
type CheckStatus string

const (
	CheckOK      CheckStatus = "ok"
	CheckWarning CheckStatus = "warning"
	CheckFailed  CheckStatus = "failed"
	CheckSkipped CheckStatus = "skipped"
)

// PreflightItem is one problem inside a check.
type PreflightItem struct {
	Severity string // "error" blocks the run; "warning" is informational
	Subject  string // Input name or path the item is about
	Message  string
}

// PreflightCheck groups the findings of one aspect of the submission.
type PreflightCheck struct {
	Name   string
	Status CheckStatus
	Detail string
	Items  []PreflightItem
}

// PreflightReport is the full checklist.
type PreflightReport struct {
	WorkflowName string
	Checks       []PreflightCheck
}

// HasErrors reports whether anything would make the run fail.
func (r *PreflightReport) HasErrors() bool {
	for _, c := range r.Checks {
		if c.Status == CheckFailed {
			return true
		}
	}
	return false
}

// Counts returns how many errors and warnings the report holds.
func (r *PreflightReport) Counts() (errCount, warnCount int) {
	for _, c := range r.Checks {
		for _, i := range c.Items {
			if i.Severity == string(wdl.SeverityError) {
				errCount++
			} else {
				warnCount++
			}
		}
	}
	return errCount, warnCount
}

// PreflightFailedError reports that preflight found blocking problems. It
// carries the report so callers can render the whole checklist instead of a
// single message.
type PreflightFailedError struct {
	Report *PreflightReport
}

func (e *PreflightFailedError) Error() string {
	errCount, _ := e.Report.Counts()
	return fmt.Sprintf("preflight found %d problem(s) that would make this run fail", errCount)
}

// PreflightUseCase validates a submission before it happens.
type PreflightUseCase struct {
	fileProvider ports.FileProvider
	health       ports.HealthChecker
}

// NewPreflightUseCase creates a new preflight use case. health may be nil,
// in which case the server check is skipped.
func NewPreflightUseCase(fp ports.FileProvider, health ports.HealthChecker) *PreflightUseCase {
	return &PreflightUseCase{fileProvider: fp, health: health}
}

// PreflightInput is the input for a preflight run.
type PreflightInput struct {
	WorkflowFile string
	InputsFile   string
	// SkipServer skips the Cromwell health check (submit does this: it is
	// about to contact the server anyway).
	SkipServer bool
	// SkipPaths skips verifying that File inputs exist.
	SkipPaths bool
	// DependenciesFile is an optional imports zip; its contents are checked
	// against the workflow's imports.
	DependenciesFile string
}

// Execute runs every check and returns the full report. It does not stop at
// the first failure: the point is to show everything that needs fixing.
func (uc *PreflightUseCase) Execute(ctx context.Context, input PreflightInput) (*PreflightReport, error) {
	if input.WorkflowFile == "" {
		return nil, application.NewInputValidationError("workflowFile", "is required")
	}

	source, err := uc.fileProvider.ReadBytes(ctx, input.WorkflowFile)
	if err != nil {
		return nil, application.NewUseCaseError("preflight", "failed to read workflow file", err)
	}

	var inputsData []byte
	if input.InputsFile != "" {
		inputsData, err = uc.fileProvider.ReadBytes(ctx, input.InputsFile)
		if err != nil {
			return nil, application.NewUseCaseError("preflight", "failed to read inputs file", err)
		}
	}

	var depsData []byte
	if input.DependenciesFile != "" {
		depsData, err = uc.fileProvider.ReadBytes(ctx, input.DependenciesFile)
		if err != nil {
			return nil, application.NewUseCaseError("preflight", "failed to read dependencies file", err)
		}
	}

	return uc.check(ctx, source, inputsData, depsData, input.SkipServer, input.SkipPaths), nil
}

// check runs the checklist over already-read sources, so callers that hold
// the bytes (submit) do not read them twice.
func (uc *PreflightUseCase) check(ctx context.Context, source, inputsData, depsData []byte, skipServer, skipPaths bool) *PreflightReport {
	report := &PreflightReport{}
	report.Checks = append(report.Checks, uc.checkServer(ctx, skipServer))

	inputsReport := wdl.CheckInputs(source, inputsData)
	report.WorkflowName = inputsReport.WorkflowName
	report.Checks = append(report.Checks, syntaxCheck(inputsReport), inputsCheck(inputsReport))

	report.Checks = append(report.Checks, uc.checkPaths(ctx, inputsReport.Files, skipPaths))
	report.Checks = append(report.Checks, dependenciesCheck(source, depsData))

	return report
}

// checkServer reports whether Cromwell is reachable and healthy.
func (uc *PreflightUseCase) checkServer(ctx context.Context, skip bool) PreflightCheck {
	check := PreflightCheck{Name: "Cromwell server"}
	if skip || uc.health == nil {
		check.Status = CheckSkipped
		check.Detail = "not checked"
		return check
	}

	status, err := uc.health.GetHealthStatus(ctx)
	switch {
	case err != nil:
		check.Status = CheckFailed
		check.Detail = "unreachable"
		check.Items = append(check.Items, PreflightItem{
			Severity: string(wdl.SeverityError),
			Message:  fmt.Sprintf("could not reach Cromwell: %v — check CROMWELL_HOST", err),
		})
	case status != nil && !status.OK:
		// Degraded subsystems are worth knowing about but do not stop a run.
		check.Status = CheckWarning
		check.Detail = "degraded"
		check.Items = append(check.Items, PreflightItem{
			Severity: string(wdl.SeverityWarning),
			Message:  fmt.Sprintf("server reports unhealthy subsystems: %v", status.UnhealthySystems),
		})
	default:
		check.Status = CheckOK
		check.Detail = "reachable"
	}
	return check
}

// syntaxCheck reports whether the WDL could be parsed locally.
func syntaxCheck(r *wdl.InputsReport) PreflightCheck {
	check := PreflightCheck{Name: "WDL syntax"}
	if r.Parsed {
		check.Status = CheckOK
		check.Detail = "workflow " + r.WorkflowName
		return check
	}
	// Never a failure: Cromwell is the authority on WDL this parser cannot read.
	check.Status = CheckWarning
	check.Detail = "could not parse locally"
	for _, f := range r.Findings {
		check.Items = append(check.Items, PreflightItem{Severity: string(f.Severity), Message: f.Message})
	}
	return check
}

// inputsCheck turns the WDL-level findings into a check.
func inputsCheck(r *wdl.InputsReport) PreflightCheck {
	check := PreflightCheck{Name: "Inputs"}
	if !r.Parsed {
		check.Status = CheckSkipped
		check.Detail = "WDL could not be parsed"
		return check
	}

	for _, f := range r.Findings {
		check.Items = append(check.Items, PreflightItem{
			Severity: string(f.Severity),
			Subject:  f.Input,
			Message:  f.Message,
		})
	}

	switch {
	case r.HasErrors():
		check.Status = CheckFailed
		check.Detail = "problems found"
	case len(check.Items) > 0:
		check.Status = CheckWarning
		check.Detail = "worth a look"
	default:
		check.Status = CheckOK
		check.Detail = "complete and well-typed"
	}
	return check
}

// checkPaths verifies that every File input points at something that exists.
// A missing file is the user's problem (error); anything else — no
// credentials, network trouble — only means we could not check (warning),
// because Cromwell may well have access this machine does not.
func (uc *PreflightUseCase) checkPaths(ctx context.Context, files []wdl.FileRef, skip bool) PreflightCheck {
	check := PreflightCheck{Name: "Input files"}
	switch {
	case skip:
		check.Status = CheckSkipped
		check.Detail = "not checked"
		return check
	case len(files) == 0:
		check.Status = CheckOK
		check.Detail = "no file inputs to check"
		return check
	}

	items := make([]*PreflightItem, len(files))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(pathCheckConcurrency)
	for i, ref := range files {
		i, ref := i, ref
		g.Go(func() error {
			if _, err := uc.fileProvider.GetSize(gctx, ref.Path); err != nil {
				if errors.Is(err, ports.ErrFileNotFound) {
					items[i] = &PreflightItem{
						Severity: string(wdl.SeverityError),
						Subject:  ref.Input,
						Message:  fmt.Sprintf("file does not exist: %s", ref.Path),
					}
					return nil
				}
				items[i] = &PreflightItem{
					Severity: string(wdl.SeverityWarning),
					Subject:  ref.Input,
					Message:  fmt.Sprintf("could not verify %s: %v", ref.Path, err),
				}
			}
			return nil
		})
	}
	_ = g.Wait() // Individual failures become items; the group itself never errors.

	// Reported in input order, not completion order.
	for _, item := range items {
		if item != nil {
			check.Items = append(check.Items, *item)
		}
	}

	switch {
	case hasSeverity(check.Items, wdl.SeverityError):
		check.Status = CheckFailed
		check.Detail = "missing files"
	case len(check.Items) > 0:
		check.Status = CheckWarning
		check.Detail = "some could not be verified"
	default:
		check.Status = CheckOK
		check.Detail = fmt.Sprintf("%d file(s) found", len(files))
	}
	return check
}

func hasSeverity(items []PreflightItem, s wdl.Severity) bool {
	for _, i := range items {
		if i.Severity == string(s) {
			return true
		}
	}
	return false
}

// dependenciesCheck verifies the imports resolve inside the dependencies zip.
// Skipped when no zip is provided (a self-contained workflow needs none).
func dependenciesCheck(source, depsData []byte) PreflightCheck {
	check := PreflightCheck{Name: "Dependencies"}
	if len(depsData) == 0 {
		check.Status = CheckSkipped
		check.Detail = "no dependencies zip"
		return check
	}

	report := wdl.CheckDependencies(source, depsData)
	for _, f := range report.Findings {
		check.Items = append(check.Items, PreflightItem{Severity: string(f.Severity), Message: f.Message})
	}

	switch {
	case report.HasErrors():
		check.Status = CheckFailed
		check.Detail = "missing imports"
	case !report.ZipRead:
		check.Status = CheckWarning
		check.Detail = "could not read the zip"
	default:
		check.Status = CheckOK
		check.Detail = fmt.Sprintf("%d file(s), all imports resolve", report.WDLFiles)
	}
	return check
}
