// Package history exposes the local run history — what was submitted from
// this machine and what it was for — to the agent.
package history

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// action is the tool action this handler answers.
const action = "history"

// defaultLimit keeps an unbounded question ("what have I been running?") from
// dumping the whole history into the conversation.
const defaultLimit = 10

// Handler answers the "history" action. It reads only what this machine
// remembers; live state comes from the Cromwell actions.
type Handler struct {
	store ports.RunHistoryReader
	host  ports.HostProvider
}

// NewHandler creates a new history Handler.
func NewHandler(store ports.RunHistoryReader, host ports.HostProvider) *Handler {
	return &Handler{store: store, host: host}
}

// Handle implements types.Handler. With a workflow_id it answers about that
// run; otherwise it lists the most recent ones, optionally filtered by query.
func (h *Handler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	host, alias := h.host.CurrentHost()

	if input.WorkflowID != "" {
		rec, err := h.store.Get(ctx, ports.RunRef{Host: host, WorkflowID: input.WorkflowID})
		if errors.Is(err, ports.ErrRunNotRemembered) {
			return types.NewErrorOutput(action, fmt.Sprintf(
				"no local record for %s on %s. Only runs submitted through pumbaa, or annotated with `pumbaa history note`, are remembered here",
				input.WorkflowID, hostLabel(host, alias))), nil
		}
		if err != nil {
			return types.NewErrorOutput(action, err.Error()), nil
		}
		return types.NewSuccessOutput(action, map[string]any{
			"host": hostLabel(host, alias),
			"run":  view(rec),
		}), nil
	}

	limit := input.PageSize
	if limit <= 0 {
		limit = defaultLimit
	}

	filter := ports.RunHistoryFilter{
		Host:   host,
		Search: input.Query,
		Status: input.Status,
		Limit:  limit,
	}
	if input.SinceDays > 0 {
		filter.Since = time.Now().AddDate(0, 0, -input.SinceDays)
	}

	records, err := h.store.List(ctx, filter)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	runs := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		runs = append(runs, view(rec))
	}

	return types.NewSuccessOutput(action, map[string]any{
		"host": hostLabel(host, alias),
		"runs": runs,
		// The statuses are the last ones seen locally, which can lag; say so
		// rather than let the model present them as current.
		"note": "statuses are the last ones recorded locally; use the status or query actions for live state",
	}), nil
}

// view renders one record, dropping what was never filled in so the model is
// not handed a page of empty fields.
func view(rec ports.RunRecord) map[string]any {
	out := map[string]any{
		"workflow_id":  rec.WorkflowID,
		"submitted_at": rec.SubmittedAt,
		"remembered":   string(rec.Origin),
	}
	optional := map[string]string{
		"name":              rec.Name,
		"description":       rec.Description,
		"last_status":       rec.LastStatus,
		"workflow_file":     rec.WorkflowFile,
		"inputs_file":       rec.InputsFile,
		"options_file":      rec.OptionsFile,
		"dependencies_file": rec.DependenciesFile,
	}
	for key, value := range optional {
		if value != "" {
			out[key] = value
		}
	}
	if len(rec.Labels) > 0 {
		out["labels"] = rec.Labels
	}
	return out
}

func hostLabel(host, alias string) string {
	if alias != "" {
		return alias
	}
	return host
}
