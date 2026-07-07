// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"sort"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	workflowdomain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// MetadataHandler handles the workflow metadata command.
type MetadataHandler struct {
	useCase   *workflow.MetadataUseCase
	presenter *presenter.Presenter
}

// NewMetadataHandler creates a new MetadataHandler.
func NewMetadataHandler(uc *workflow.MetadataUseCase, p *presenter.Presenter) *MetadataHandler {
	return &MetadataHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for workflow metadata retrieval.
func (h *MetadataHandler) Command() *cli.Command {
	return &cli.Command{
		Name:      "metadata",
		Aliases:   []string{"m", "meta"},
		Usage:     "Get workflow metadata and display status",
		ArgsUsage: "<workflow-id>",
		Action:    h.handle,
	}
}

func (h *MetadataHandler) handle(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Workflow ID is required")
		return cli.Exit("workflow ID required", 1)
	}

	ctx := context.Background()
	workflowID := c.Args().First()

	input := workflow.MetadataInput{
		WorkflowID: workflowID,
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to get workflow metadata: %v", err)
		return err
	}

	h.displayMetadata(output)

	return nil
}

func (h *MetadataHandler) displayMetadata(wf *workflowdomain.Workflow) {
	// Workflow overview
	h.presenter.Title("Workflow Details")
	h.presenter.KeyValue("ID", wf.ID)
	h.presenter.KeyValue("Name", wf.Name)
	h.presenter.KeyValue("Status", h.presenter.StatusColor(string(wf.Status)))
	h.presenter.KeyValue("Start", h.presenter.FormatTime(wf.Start))
	h.presenter.KeyValue("End", h.presenter.FormatTime(wf.End))
	h.presenter.KeyValue("Duration", h.presenter.FormatDuration(wf.Duration()))

	// Labels
	if len(wf.Labels) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Labels")
		for k, v := range wf.Labels {
			h.presenter.KeyValue(k, v)
		}
	}

	// Calls summary
	if len(wf.Calls) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Calls")

		// Flatten calls map into slice for sorting
		type callEntry struct {
			name string
			call workflowdomain.Call
		}
		var entries []callEntry
		for callName, calls := range wf.Calls {
			for _, call := range calls {
				entries = append(entries, callEntry{name: callName, call: call})
			}
		}

		// Sort by start time
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].call.Start.Before(entries[j].call.Start)
		})

		table := h.presenter.NewTable([]string{"Task", "Status", "Duration", "Shard", "Attempt"})

		for _, entry := range entries {
			shard := "-"
			if entry.call.ShardIndex >= 0 {
				shard = string(rune('0' + entry.call.ShardIndex))
			}

			duration := h.calculateCallDuration(entry.call)

			table.Append([]string{
				entry.name,
				h.presenter.StatusColor(string(entry.call.Status)),
				h.presenter.FormatDuration(duration),
				shard,
				string(rune('0' + entry.call.Attempt)),
			})
		}

		table.Render()
	}

	// Failures
	if len(wf.Failures) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Failures")
		for _, failure := range wf.Failures {
			h.displayFailure(failure)
		}
	}
}

func (h *MetadataHandler) calculateCallDuration(call workflowdomain.Call) time.Duration {
	if call.Start.IsZero() {
		return 0
	}
	end := call.End
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(call.Start)
}

func (h *MetadataHandler) displayFailure(failure workflowdomain.Failure) {
	h.presenter.Error(failure.Message)
	for _, cause := range failure.CausedBy {
		h.displayFailure(cause)
	}
}
