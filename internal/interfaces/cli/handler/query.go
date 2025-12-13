// Package handler provides CLI command handlers.
package handler

import (
	"context"

	"github.com/lmtani/pumbaa/internal/application/workflow/query"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
	"github.com/urfave/cli/v2"
)

// QueryHandler handles the workflow query command.
type QueryHandler struct {
	useCase   *query.UseCase
	presenter *presenter.Presenter
}

// NewQueryHandler creates a new QueryHandler.
func NewQueryHandler(uc *query.UseCase, p *presenter.Presenter) *QueryHandler {
	return &QueryHandler{
		useCase:   uc,
		presenter: p,
	}
}

// Command returns the CLI command for querying workflows.
func (h *QueryHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "query",
		Aliases: []string{"q", "list"},
		Usage:   "Query and list workflows",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   "[optional] Filter by workflow name",
			},
			&cli.StringSliceFlag{
				Name:    "status",
				Aliases: []string{"s"},
				Usage:   "[optional] Filter by status (can be specified multiple times)",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "[optional] Maximum number of results",
				Value:   20,
			},
		},
		Action: h.handle,
	}
}

func (h *QueryHandler) handle(c *cli.Context) error {
	ctx := context.Background()

	input := query.Input{
		Name:     c.String("name"),
		Status:   c.StringSlice("status"),
		PageSize: c.Int("limit"),
	}

	output, err := h.useCase.Execute(ctx, input)
	if err != nil {
		h.presenter.Error("Failed to query workflows: %v", err)
		return err
	}

	if len(output.Workflows) == 0 {
		h.presenter.Info("No workflows found matching the criteria")
		return nil
	}

	h.presenter.Title("Workflows")
	h.presenter.Info("Found %d workflow(s)", output.TotalCount)
	h.presenter.Newline()

	table := h.presenter.NewTable([]string{"ID", "Name", "Status", "Submitted"})

	for _, wf := range output.Workflows {
		table.Append([]string{
			wf.ID, // Full UUID
			wf.Name,
			h.presenter.StatusColor(wf.Status),
			h.presenter.FormatTime(wf.SubmittedAt),
		})
	}

	table.Render()

	return nil
}
