// Package handler provides CLI command handlers.
package handler

import (
	"context"
	"sort"

	"github.com/lmtani/pumbaa/internal/application/workflow/metadata"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
	"github.com/urfave/cli/v2"
)

// MetadataHandler handles the workflow metadata command.
type MetadataHandler struct {
	useCase   *metadata.UseCase
	presenter *presenter.Presenter
}

// NewMetadataHandler creates a new MetadataHandler.
func NewMetadataHandler(uc *metadata.UseCase, p *presenter.Presenter) *MetadataHandler {
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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "[optional] Show detailed call information",
			},
		},
		Action: h.handle,
	}
}

func (h *MetadataHandler) handle(c *cli.Context) error {
	if c.NArg() < 1 {
		h.presenter.Error("Workflow ID is required")
		return cli.Exit("workflow ID required", 1)
	}

	ctx := context.Background()
	workflowID := c.Args().First()

	input := metadata.Input{
		WorkflowID: workflowID,
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to get workflow metadata: %v", err)
		return err
	}

	h.displayMetadata(output, c.Bool("verbose"))

	return nil
}

func (h *MetadataHandler) displayMetadata(m *metadata.Output, verbose bool) {
	// Workflow overview
	h.presenter.Title("Workflow Details")
	h.presenter.KeyValue("ID", m.ID)
	h.presenter.KeyValue("Name", m.Name)
	h.presenter.KeyValue("Status", h.presenter.StatusColor(m.Status))
	h.presenter.KeyValue("Start", h.presenter.FormatTime(m.Start))
	h.presenter.KeyValue("End", h.presenter.FormatTime(m.End))
	h.presenter.KeyValue("Duration", h.presenter.FormatDuration(m.Duration))

	// Labels
	if len(m.Labels) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Labels")
		for k, v := range m.Labels {
			h.presenter.KeyValue(k, v)
		}
	}

	// Calls summary
	if len(m.Calls) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Calls")

		// Sort calls by start time
		calls := m.Calls
		sort.Slice(calls, func(i, j int) bool {
			return calls[i].Start.Before(calls[j].Start)
		})

		table := h.presenter.NewTable([]string{"Task", "Status", "Duration", "Shard", "Attempt"})

		for _, call := range calls {
			shard := "-"
			if call.ShardIndex >= 0 {
				shard = string(rune('0' + call.ShardIndex))
			}

			table.Append([]string{
				call.Name,
				h.presenter.StatusColor(call.Status),
				h.presenter.FormatDuration(call.Duration),
				shard,
				string(rune('0' + call.Attempt)),
			})
		}

		table.Render()
	}

	// Failures
	if len(m.Failures) > 0 {
		h.presenter.Newline()
		h.presenter.Title("Failures")
		for _, failure := range m.Failures {
			h.presenter.Error(failure)
		}
	}
}
