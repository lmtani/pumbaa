package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// descriptionWidth keeps the listing readable; the full text is one
// `history show` away.
const descriptionWidth = 46

// HistoryHandler serves the local run history.
type HistoryHandler struct {
	useCase   *workflow.RunHistoryUseCase
	presenter *presenter.Presenter
}

// NewHistoryHandler creates a new HistoryHandler.
func NewHistoryHandler(uc *workflow.RunHistoryUseCase, p *presenter.Presenter) *HistoryHandler {
	return &HistoryHandler{useCase: uc, presenter: p}
}

// Command returns the CLI command for the run history.
func (h *HistoryHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "Browse the local history of submitted runs",
		Description: `Runs submitted through pumbaa are remembered locally, with whatever
description was given at submission time. The record outlives the server:
it still answers what a run was for after Cromwell has forgotten it.

Notes can be attached to any run, not only to ones submitted from here.`,
		Flags:  listHistoryFlags(),
		Action: h.list,
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List remembered runs",
				Flags:  listHistoryFlags(),
				Action: h.list,
			},
			{
				Name:      "show",
				Usage:     "Show everything remembered about a run",
				ArgsUsage: "<workflow-id>",
				Action:    h.show,
			},
			{
				Name:      "note",
				Usage:     "Write (or rewrite) the description of a run",
				ArgsUsage: "<workflow-id> <description>",
				Action:    h.note,
			},
			{
				Name:      "forget",
				Usage:     "Drop a run from the local history",
				ArgsUsage: "<workflow-id>",
				Action:    h.forget,
			},
			{
				Name:  "prune",
				Usage: "Drop runs older than a cutoff",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "older-than",
						Usage:    "[required] Age cutoff, e.g. 90d, 12w, 720h",
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "yes",
						Usage: "[optional] Do not ask for confirmation",
					},
				},
				Action: h.prune,
			},
		},
	}
}

func listHistoryFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "[optional] List runs of every host, not just the active one",
		},
		&cli.StringFlag{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "[optional] Match against the description, name or ID",
		},
		&cli.StringFlag{
			Name:  "since",
			Usage: "[optional] Only runs submitted since, e.g. 7d, 24h or 2026-07-01",
		},
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "[optional] Maximum number of runs",
			Value:   20,
		},
		&cli.BoolFlag{
			Name:  "no-refresh",
			Usage: "[optional] Do not ask the server for current statuses",
		},
		&cli.BoolFlag{
			Name:  "json",
			Usage: "[optional] Print the records as JSON",
		},
	}
}

func (h *HistoryHandler) list(c *cli.Context) error {
	since, err := parseSince(c.String("since"))
	if err != nil {
		return err
	}

	out, err := h.useCase.List(c.Context, workflow.ListRunHistoryInput{
		AllHosts: c.Bool("all"),
		Search:   c.String("search"),
		Since:    since,
		Limit:    c.Int("limit"),
		Refresh:  !c.Bool("no-refresh"),
	})
	if err != nil {
		h.presenter.Error("Failed to read the local history: %v", err)
		return err
	}

	if c.Bool("json") {
		return printJSON(out.Entries)
	}

	if len(out.Entries) == 0 {
		h.presenter.Info("Nothing remembered yet%s.", scopeSuffix(c, out.Host))
		h.presenter.Newline()
		h.presenter.Info("Runs are remembered when submitted through pumbaa; add a note to an")
		h.presenter.Info("existing run with: pumbaa history note <workflow-id> \"what it was for\"")
		if !c.Bool("all") {
			h.presenter.Info("Other hosts may have records: pumbaa history --all")
		}
		return nil
	}

	h.presenter.Title("Run history")
	headers := []string{"Status", "ID", "Name", "Submitted", "Description"}
	if c.Bool("all") {
		headers = append([]string{"Host"}, headers...)
	}
	table := h.presenter.NewTable(headers)

	anyForgotten := false
	for _, entry := range out.Entries {
		if entry.Forgotten {
			anyForgotten = true
		}
		row := []string{
			h.renderStatus(entry),
			entry.Record.WorkflowID,
			entry.Record.Name,
			entry.Record.SubmittedAt.Local().Format("2006-01-02 15:04"),
			truncate(entry.Record.Description, descriptionWidth),
		}
		if c.Bool("all") {
			row = append([]string{hostLabel(entry.Record)}, row...)
		}
		_ = table.Append(row)
	}
	_ = table.Render()

	h.presenter.Newline()
	if anyForgotten {
		h.presenter.Info("* the server no longer knows this run — only the local record remains")
	}
	h.reportRefresh(out.RefreshError, out.Host, c.Bool("all"), c.Bool("no-refresh"))
	return nil
}

func (h *HistoryHandler) show(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: pumbaa history show <workflow-id>")
	}
	id := c.Args().First()

	out, err := h.useCase.Show(c.Context, id, true)
	if err != nil {
		if errors.Is(err, ports.ErrRunNotRemembered) {
			h.presenter.Info("No local record for %s.", id)
			h.presenter.Info("Write one with: pumbaa history note %s \"what it was for\"", id)
			return nil
		}
		h.presenter.Error("Failed to read the local history: %v", err)
		return err
	}

	rec := out.Entry.Record
	h.presenter.Title("Run " + rec.WorkflowID)
	if rec.Description != "" {
		h.presenter.KeyValue("Description", rec.Description)
	}
	h.presenter.KeyValue("Status", h.renderStatus(out.Entry))
	h.presenter.KeyValue("Workflow", orDash(rec.Name))
	h.presenter.KeyValue("Host", hostLabel(rec))
	h.presenter.KeyValue("Submitted", rec.SubmittedAt.Local().Format(time.RFC1123))
	h.presenter.KeyValue("Remembered", originLabel(rec.Origin))
	if rec.WorkflowFile != "" {
		h.presenter.KeyValue("WDL", rec.WorkflowFile)
	}
	if rec.InputsFile != "" {
		h.presenter.KeyValue("Inputs", rec.InputsFile)
	}
	if rec.OptionsFile != "" {
		h.presenter.KeyValue("Options", rec.OptionsFile)
	}
	if rec.DependenciesFile != "" {
		h.presenter.KeyValue("Dependencies", rec.DependenciesFile)
	}
	if len(rec.Labels) > 0 {
		h.presenter.KeyValue("Labels", formatLabels(rec.Labels))
	}
	if out.Entry.Forgotten {
		h.presenter.Newline()
		h.presenter.Info("The server no longer knows this run; the status above is the last one seen.")
	}
	h.reportRefresh(out.RefreshError, rec.Host, false, false)
	return nil
}

func (h *HistoryHandler) note(c *cli.Context) error {
	if c.NArg() < 2 {
		return fmt.Errorf("usage: pumbaa history note <workflow-id> <description>")
	}
	id := c.Args().First()
	description := strings.Join(c.Args().Tail(), " ")

	if err := h.useCase.Annotate(c.Context, id, description); err != nil {
		h.presenter.Error("Failed to save the description: %v", err)
		return err
	}
	h.presenter.Success("Saved.")
	h.presenter.KeyValue(id, description)
	return nil
}

func (h *HistoryHandler) forget(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: pumbaa history forget <workflow-id>")
	}
	id := c.Args().First()

	if err := h.useCase.Forget(c.Context, id); err != nil {
		if errors.Is(err, ports.ErrRunNotRemembered) {
			h.presenter.Info("No local record for %s.", id)
			return nil
		}
		h.presenter.Error("Failed to forget the run: %v", err)
		return err
	}
	h.presenter.Success("Forgot %s.", id)
	return nil
}

func (h *HistoryHandler) prune(c *cli.Context) error {
	age, err := parseAge(c.String("older-than"))
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-age)

	if !c.Bool("yes") {
		h.presenter.Warning("This deletes every local record of runs submitted before %s.", cutoff.Local().Format("2006-01-02"))
		h.presenter.Info("Re-run with --yes to confirm.")
		return nil
	}

	removed, err := h.useCase.Prune(c.Context, cutoff)
	if err != nil {
		h.presenter.Error("Failed to prune the local history: %v", err)
		return err
	}
	h.presenter.Success("Forgot %d run(s) submitted before %s.", removed, cutoff.Local().Format("2006-01-02"))
	return nil
}

// renderStatus colours the status and marks the runs whose only remaining
// trace is the local record.
func (h *HistoryHandler) renderStatus(entry workflow.RunHistoryEntry) string {
	status := entry.Status
	if status == "" {
		status = "-"
	}
	if entry.Forgotten {
		return h.presenter.StatusColor(status) + "*"
	}
	return h.presenter.StatusColor(status)
}

// reportRefresh explains a listing whose statuses are the remembered ones.
func (h *HistoryHandler) reportRefresh(err error, host string, allHosts, skipped bool) {
	switch {
	case allHosts:
		h.presenter.Info("Statuses are the last known: listing spans hosts, so none was queried.")
	case skipped:
		return
	case err != nil:
		h.presenter.Info("Statuses are the last known: %s could not be reached.", host)
	}
}

func scopeSuffix(c *cli.Context, host string) string {
	if c.Bool("all") {
		return ""
	}
	return " for " + host
}

func hostLabel(rec ports.RunRecord) string {
	if rec.HostAlias != "" {
		return rec.HostAlias
	}
	return rec.Host
}

func originLabel(origin ports.RunOrigin) string {
	if origin == ports.OriginSubmit {
		return "submitted from this machine"
	}
	return "annotated here"
}

func formatLabels(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ", ")
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-1]) + "…"
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// parseSince turns a friendly cutoff into an instant. It takes an age
// ("7d", "24h") or a date ("2026-07-01"), because both are natural answers to
// "since when".
func parseSince(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if date, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return date, nil
	}
	age, err := parseAge(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since %q: use an age like 7d or a date like 2026-07-01", value)
	}
	return time.Now().Add(-age), nil
}

// parseAge extends Go durations with the day and week units, which are the
// ones that make sense for a history.
func parseAge(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty age")
	}

	unit := value[len(value)-1]
	if unit == 'd' || unit == 'w' {
		count, err := strconv.Atoi(value[:len(value)-1])
		if err != nil || count < 0 {
			return 0, fmt.Errorf("invalid age %q", value)
		}
		days := count
		if unit == 'w' {
			days = count * 7
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	age, err := time.ParseDuration(value)
	if err != nil || age < 0 {
		return 0, fmt.Errorf("invalid age %q: use 90d, 12w or 720h", value)
	}
	return age, nil
}
