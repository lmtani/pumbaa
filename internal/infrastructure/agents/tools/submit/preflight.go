package submit

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
	"github.com/lmtani/pumbaa/pkg/wdl"
)

// pathCheckConcurrency bounds the parallel existence checks.
const pathCheckConcurrency = 8

// PreflightHandler handles the "preflight" action: it checks an inputs JSON
// against a WDL — required inputs present, well-typed, and their file paths
// actually existing — so a broken submission is caught in the chat instead
// of failing on Cromwell minutes later.
//
// Unlike the CLI preflight, it does not check server reachability: the agent
// already talks to Cromwell for other actions, so an unreachable server
// would surface there.
type PreflightHandler struct {
	fileProvider ports.FileProvider
}

// NewPreflightHandler creates a new PreflightHandler.
func NewPreflightHandler(fp ports.FileProvider) *PreflightHandler {
	return &PreflightHandler{fileProvider: fp}
}

// Handle implements types.Handler.
func (h *PreflightHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	const action = "preflight"
	if input.WorkflowFile == "" {
		return types.NewErrorOutput(action, "workflow_file is required (a .wdl path in the working directory)"), nil
	}

	source, err := readWorkingDirFile(input.WorkflowFile)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	var inputsData []byte
	if input.InputsFile != "" {
		inputsData, err = readWorkingDirFile(input.InputsFile)
		if err != nil {
			return types.NewErrorOutput(action, err.Error()), nil
		}
	}

	report := wdl.CheckInputs(source, inputsData)

	findings := make([]map[string]any, 0, len(report.Findings))
	for _, f := range report.Findings {
		entry := map[string]any{"severity": string(f.Severity), "message": f.Message}
		if f.Input != "" {
			entry["input"] = f.Input
		}
		findings = append(findings, entry)
	}

	missing, unverifiable := h.checkPaths(ctx, report.Files)

	ready := !report.HasErrors() && len(missing) == 0
	data := map[string]any{
		"workflow":        report.WorkflowName,
		"ready":           ready,
		"parsed":          report.Parsed,
		"input_findings":  findings,
		"files_checked":   len(report.Files),
		"missing_files":   missing,
		"unverified_file": unverifiable,
	}
	if ready {
		data["hint"] = "looks ready; submit with the CLI: pumbaa workflow submit"
	} else {
		data["hint"] = "fix the errors above before submitting"
	}
	return types.NewSuccessOutput(action, data), nil
}

// checkPaths verifies that every File input exists. A path known to be
// missing is the user's problem (missing); anything else — no credentials,
// network — only means it could not be checked (unverifiable), since
// Cromwell may reach what this machine cannot.
func (h *PreflightHandler) checkPaths(ctx context.Context, files []wdl.FileRef) (missing, unverifiable []string) {
	if h.fileProvider == nil || len(files) == 0 {
		return nil, nil
	}

	type result struct{ missing, unverifiable string }
	results := make([]result, len(files))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(pathCheckConcurrency)
	for i, ref := range files {
		i, ref := i, ref
		g.Go(func() error {
			if _, err := h.fileProvider.GetSize(gctx, ref.Path); err != nil {
				if errors.Is(err, ports.ErrFileNotFound) {
					results[i].missing = fmt.Sprintf("%s: %s", ref.Input, ref.Path)
				} else {
					results[i].unverifiable = fmt.Sprintf("%s: %s", ref.Input, ref.Path)
				}
			}
			return nil
		})
	}
	_ = g.Wait()

	for _, r := range results {
		if r.missing != "" {
			missing = append(missing, r.missing)
		}
		if r.unverifiable != "" {
			unverifiable = append(unverifiable, r.unverifiable)
		}
	}
	return missing, unverifiable
}
