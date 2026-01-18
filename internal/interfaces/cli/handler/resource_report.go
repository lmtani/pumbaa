// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// ResourceReportHandler handles the workflow resource report command.
type ResourceReportHandler struct {
	useCase   *workflow.ResourceReportUseCase
	presenter *presenter.Presenter
}

// NewResourceReportHandler creates a new ResourceReportHandler.
func NewResourceReportHandler(uc *workflow.ResourceReportUseCase, p *presenter.Presenter) *ResourceReportHandler {
	return &ResourceReportHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for resource report generation.
func (h *ResourceReportHandler) Command() *cli.Command {
	return &cli.Command{
		Name:      "resource-report",
		Aliases:   []string{"rr", "resources"},
		Usage:     "Generate resource usage report for all tasks in a workflow",
		ArgsUsage: "<workflow-id>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "concurrency",
				Aliases: []string{"c"},
				Usage:   "Number of concurrent workers for fetching monitoring logs",
				Value:   5,
			},
		},
		Action: h.handle,
	}
}

func (h *ResourceReportHandler) handle(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Workflow ID is required")
		return cli.Exit("workflow ID required", 1)
	}

	ctx := context.Background()
	workflowID := c.Args().First()
	concurrency := c.Int("concurrency")

	input := workflow.ResourceReportInput{
		WorkflowID:  workflowID,
		Concurrency: concurrency,
	}

	h.presenter.Info("Fetching workflow metadata...")

	var lastPrinted int64
	progress := func(completed, total int, currentTask string) {
		// Only print every 5% progress to avoid too much output
		pct := int64(float64(completed) / float64(total) * 100)
		if pct >= atomic.LoadInt64(&lastPrinted)+5 || completed == total {
			atomic.StoreInt64(&lastPrinted, pct)
			h.presenter.Print("\rProcessing tasks: %d/%d (%d%%) - %s", completed, total, pct, currentTask)
		}
	}

	output, err := h.useCase.ExecuteWithProgress(ctx, input, progress)
	if err != nil {
		h.presenter.Error("\nFailed to generate resource report: %v", err)
		return err
	}

	h.presenter.Println("") // New line after progress
	h.displayResults(output)
	return nil
}

func (h *ResourceReportHandler) displayResults(output *workflow.ResourceReportOutput) {
	h.presenter.Newline()
	h.presenter.Title("Resource Report")
	h.presenter.KeyValue("Workflow ID", output.WorkflowID)
	h.presenter.KeyValue("Workflow Name", output.WorkflowName)
	h.presenter.KeyValue("Tasks Analyzed", fmt.Sprintf("%d", len(output.Tasks)))
	h.presenter.KeyValue("Output File", output.OutputFile)
	h.presenter.Newline()

	if len(output.Tasks) == 0 {
		h.presenter.Info("No tasks with monitoring logs found")
		return
	}

	// Display summary table
	table := h.presenter.NewTable([]string{"Task", "Shard", "CPU", "Mem Req", "Disk Req", "Type", "Input", "CPU Mean", "Mem Peak", "Disk Peak", "Status"})

	var errorCount int
	for _, task := range output.Tasks {
		status := "OK"
		if task.Error != "" {
			status = "Error"
			errorCount++
		}

		shardStr := "-"
		if task.ShardIndex >= 0 {
			shardStr = fmt.Sprintf("%d", task.ShardIndex)
		}

		table.Append([]string{
			task.TaskName,
			shardStr,
			task.CPURequest,
			formatBytes(task.MemoryRequestBytes),
			formatBytes(task.DiskSizeRequestBytes),
			task.DiskType,
			formatBytes(task.TotalInputBytes),
			fmt.Sprintf("%.1f%%", task.CPUMean),
			fmt.Sprintf("%.0f MB", task.MemoryPeakMB),
			formatBytes(task.DiskPeakBytes),
			status,
		})
	}

	table.Render()

	if errorCount > 0 {
		h.presenter.Newline()
		h.presenter.Warning("%d task(s) had errors reading monitoring logs", errorCount)
	}

	h.presenter.Newline()
	h.presenter.Success("Report saved to: %s", output.OutputFile)
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
