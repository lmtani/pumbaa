package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	workflowdomain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// DiffHandler handles the workflow diff command.
type DiffHandler struct {
	useCase   *workflow.CompareUseCase
	presenter *presenter.Presenter
}

// NewDiffHandler creates a new DiffHandler.
func NewDiffHandler(uc *workflow.CompareUseCase, p *presenter.Presenter) *DiffHandler {
	return &DiffHandler{useCase: uc, presenter: p}
}

// Command returns the CLI command for comparing two workflow runs.
func (h *DiffHandler) Command() *cli.Command {
	return &cli.Command{
		Name:      "diff",
		Aliases:   []string{"compare"},
		Usage:     "Compare two workflow runs (inputs, options, source, tasks)",
		ArgsUsage: "<workflow-id-a> <workflow-id-b>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "[optional] Output the diff as JSON",
			},
			&cli.BoolFlag{
				Name:  "no-cache-resolve",
				Usage: "[optional] Do not fetch cache sources to recover real metrics of cache-hit tasks",
			},
		},
		Action: h.handle,
	}
}

func (h *DiffHandler) handle(c *cli.Context) error {
	if c.NArg() < 2 {
		h.presenter.Error("Two workflow IDs are required: pumbaa workflow diff <id-a> <id-b>")
		return cli.Exit("two workflow IDs required", 1)
	}

	ctx := context.Background()
	input := workflow.CompareInput{
		WorkflowIDA:  c.Args().Get(0),
		WorkflowIDB:  c.Args().Get(1),
		ResolveCache: !c.Bool("no-cache-resolve"),
	}

	diff, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to compare workflows: %v", err)
		return err
	}

	if c.Bool("json") {
		return h.printJSON(diff)
	}

	h.display(diff)
	return nil
}

func (h *DiffHandler) printJSON(diff *workflowdomain.RunDiff) error {
	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		h.presenter.Error("Failed to encode diff: %v", err)
		return err
	}
	h.presenter.Println(string(data))
	return nil
}

func (h *DiffHandler) display(diff *workflowdomain.RunDiff) {
	h.presenter.Title("Workflow Diff")
	h.presenter.Print("  %s  %s  %s  %s\n",
		color.New(color.Bold).Sprint("A:"),
		labelOrDash(diff.NameA), shortID(diff.IDA),
		h.presenter.StatusColor(string(diff.StatusA)))
	h.presenter.Print("  %s  %s  %s  %s\n",
		color.New(color.Bold).Sprint("B:"),
		labelOrDash(diff.NameB), shortID(diff.IDB),
		h.presenter.StatusColor(string(diff.StatusB)))

	if diff.NameMismatch {
		h.presenter.Warning("Workflow names differ — comparing runs of different workflows")
	}
	if !diff.HasDifferences() {
		h.presenter.Newline()
		h.presenter.Success("No differences found")
		return
	}

	h.displayKeySection("Inputs", diff.Inputs)
	h.displayKeySection("Options", diff.Options)
	h.displaySource(diff)
	h.displayTasks(diff)
}

func (h *DiffHandler) displayKeySection(title string, diffs []workflowdomain.KeyDiff) {
	h.presenter.Newline()
	if len(diffs) == 0 {
		h.presenter.Title(title + " (no changes)")
		return
	}
	h.presenter.Title(fmt.Sprintf("%s (%d changed)", title, len(diffs)))
	for _, kd := range diffs {
		switch kd.Kind {
		case workflowdomain.ChangeAdded:
			h.presenter.Print("  %s %s  %s\n", color.GreenString("+"), kd.Key, valuePreview(kd.ValueB))
		case workflowdomain.ChangeRemoved:
			h.presenter.Print("  %s %s  %s\n", color.RedString("-"), kd.Key, valuePreview(kd.ValueA))
		default:
			h.presenter.Print("  %s %s  %s → %s\n", color.YellowString("~"), kd.Key,
				valuePreview(kd.ValueA), valuePreview(kd.ValueB))
		}
	}
}

func (h *DiffHandler) displaySource(diff *workflowdomain.RunDiff) {
	h.presenter.Newline()
	if !diff.SourceChanged {
		h.presenter.Title("Source (unchanged)")
		return
	}
	h.presenter.Title("Source")
	h.presenter.Print("  %s WDL source changed (%d → %d lines)\n",
		color.YellowString("~"), diff.SourceLinesA, diff.SourceLinesB)
}

func (h *DiffHandler) displayTasks(diff *workflowdomain.RunDiff) {
	h.presenter.Newline()
	if len(diff.Tasks) == 0 {
		h.presenter.Title(fmt.Sprintf("Tasks (no changes, %d compared)", diff.TotalTasks))
		return
	}
	h.presenter.Title(fmt.Sprintf("Tasks (%d of %d changed)", len(diff.Tasks), diff.TotalTasks))

	for _, td := range diff.Tasks {
		switch td.Kind {
		case workflowdomain.ChangeAdded:
			h.presenter.Print("  %s %s  %s  (only in B)\n",
				color.GreenString("+"), td.Name, h.presenter.StatusColor(td.StatusB))
		case workflowdomain.ChangeRemoved:
			h.presenter.Print("  %s %s  %s  (only in A)\n",
				color.RedString("-"), td.Name, h.presenter.StatusColor(td.StatusA))
		default:
			h.presenter.Print("  %s %s\n", color.YellowString("~"), td.Name)
			for _, line := range h.taskChangeLines(td) {
				h.presenter.Print("      %s\n", line)
			}
		}
	}
}

// taskChangeLines describes the specific changes of a modified task.
func (h *DiffHandler) taskChangeLines(td workflowdomain.TaskDiff) []string {
	var lines []string
	if td.StatusChanged() {
		lines = append(lines, fmt.Sprintf("status:   %s → %s",
			h.presenter.StatusColor(td.StatusA), h.presenter.StatusColor(td.StatusB)))
	}
	if td.DockerChanged() {
		lines = append(lines, fmt.Sprintf("docker:   %s → %s",
			labelOrDash(td.DockerA), labelOrDash(td.DockerB)))
	}
	if td.ShardsChanged() {
		lines = append(lines, fmt.Sprintf("shards:   %d → %d", td.ShardsA, td.ShardsB))
	}
	if td.AttemptsA != td.AttemptsB {
		lines = append(lines, fmt.Sprintf("attempts: %d → %d", td.AttemptsA, td.AttemptsB))
	}
	if td.DurationChangedSignificantly() {
		lines = append(lines, fmt.Sprintf("duration: %s → %s  (%s)",
			h.presenter.FormatDuration(td.DurationA),
			h.presenter.FormatDuration(td.DurationB),
			durationVerdict(td.DurationRatio())))
	}
	if note := cacheNote(td); note != "" {
		lines = append(lines, note)
	}
	return lines
}

// cacheNote describes the cache provenance of a task's two sides: which side's
// metrics were recovered from a source run, which had an unresolved cache hit,
// and which was a subworkflow whose work was served from cache. In the latter
// two cases the duration was not compared.
func cacheNote(td workflowdomain.TaskDiff) string {
	var parts []string
	if a := sideCacheNote("A", td.RecoveredA, td.CacheSourceA, td.UnresolvedCacheA, td.SubworkflowCachedA); a != "" {
		parts = append(parts, a)
	}
	if b := sideCacheNote("B", td.RecoveredB, td.CacheSourceB, td.UnresolvedCacheB, td.SubworkflowCachedB); b != "" {
		parts = append(parts, b)
	}
	if len(parts) == 0 {
		return ""
	}
	return "cache:    " + strings.Join(parts, ", ")
}

func sideCacheNote(side string, recovered bool, source string, unresolved, subworkflowCached bool) string {
	switch {
	case recovered:
		return side + " recovered from " + shortID(source)
	case subworkflowCached:
		return side + " subworkflow served from cache"
	case unresolved:
		return side + " cache hit (source unresolved)"
	}
	return ""
}

func durationVerdict(ratio float64) string {
	if ratio > 1 {
		return color.RedString("%.1f× slower", ratio)
	}
	if ratio > 0 {
		return color.GreenString("%.1f× faster", 1/ratio)
	}
	return "changed"
}

func valuePreview(v string) string {
	const maxLen = 60
	if v == "" {
		return color.HiBlackString("(absent)")
	}
	r := []rune(v)
	if len(r) > maxLen {
		return string(r[:maxLen-1]) + "…"
	}
	return v
}

func labelOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func shortID(id string) string {
	if len(id) > 8 {
		return color.HiBlackString(id[:8])
	}
	return color.HiBlackString(id)
}
