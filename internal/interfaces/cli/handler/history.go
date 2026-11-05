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
	exporter  *workflow.ExportRunUseCase
	presenter *presenter.Presenter
}

// NewHistoryHandler creates a new HistoryHandler.
func NewHistoryHandler(uc *workflow.RunHistoryUseCase, exporter *workflow.ExportRunUseCase, p *presenter.Presenter) *HistoryHandler {
	return &HistoryHandler{useCase: uc, exporter: exporter, presenter: p}
}

// Command returns the CLI command for the run history.
func (h *HistoryHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "Browse the local history of submitted runs",
		Description: `Runs submitted through pumbaa are remembered locally, with whatever
description was given at submission time and the files they were submitted
with. The record outlives the server: it still answers what a run was for
after Cromwell has forgotten it, and "export" writes the files back out so
the run can be submitted again — here, or by whoever the directory is sent to.

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
				Name:      "export",
				Usage:     "Write a run's files to a directory, ready to submit again",
				ArgsUsage: "<workflow-id> [dir]",
				Description: `The files kept at submission time are written as workflow.wdl, inputs.json,
options.json and dependencies.zip, with labels.json and a manifest.json saying
where each came from. A run nothing was kept for is read from the server
instead — exact for the workflow and inputs, but without the dependencies
zip, which Cromwell does not keep.`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "out",
						Aliases: []string{"o"},
						Usage:   "[optional] Directory to write to, also the second argument (default: ./<workflow-id>)",
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "[optional] Replace files already in the directory",
					},
					&cli.BoolFlag{
						Name:  "strip-secrets",
						Usage: "[optional] Drop credential options (service account JSON, tokens) before sharing",
					},
				},
				Action: h.export,
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
	headers := []string{"Status", "ID", "Name", "Submitted", "Files", "Description"}
	if c.Bool("all") {
		headers = append([]string{"Host"}, headers...)
	}
	table := h.presenter.NewTable(headers)

	anyForgotten, anyKept := false, false
	for _, entry := range out.Entries {
		if entry.Forgotten {
			anyForgotten = true
		}
		kept := ""
		if len(entry.Record.Files) > 0 {
			kept, anyKept = "✓", true
		}
		row := []string{
			h.renderStatus(entry),
			entry.Record.WorkflowID,
			entry.Record.Name,
			entry.Record.SubmittedAt.Local().Format("2006-01-02 15:04"),
			kept,
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
	if anyKept {
		h.presenter.Info("✓ files kept: pumbaa history export <workflow-id> writes them out, ready to submit again")
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
	if len(rec.Files) > 0 {
		var total int64
		parts := make([]string, 0, len(rec.Files))
		for _, f := range rec.Files {
			parts = append(parts, fmt.Sprintf("%s (%s)", f.Name, workflow.FormatBytes(f.Size)))
			total += f.Size
		}
		h.presenter.KeyValue("Kept files", strings.Join(parts, ", "))
		h.presenter.Newline()
		h.presenter.Info("Submit it again, or hand it to someone: pumbaa history export %s", rec.WorkflowID)
	} else if rec.Origin == ports.OriginSubmit {
		h.presenter.KeyValue("Kept files", "none")
	}
	if out.Entry.Forgotten {
		h.presenter.Newline()
		h.presenter.Info("The server no longer knows this run; the status above is the last one seen.")
	}
	h.reportRefresh(out.RefreshError, rec.Host, false, false)
	return nil
}

func (h *HistoryHandler) export(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: pumbaa history export <workflow-id> [dir]")
	}
	id := c.Args().First()
	// The directory reads naturally after the id, where a flag would not be
	// parsed any more; --out stays for the flags-first habit.
	dir := c.Args().Get(1)
	if dir == "" {
		dir = c.String("out")
	}
	if dir == "" {
		dir = id
	}

	out, err := h.exporter.Execute(c.Context, workflow.ExportRunInput{
		WorkflowID:   id,
		Dir:          dir,
		Overwrite:    c.Bool("force"),
		StripSecrets: c.Bool("strip-secrets"),
	})
	if err != nil {
		if errors.Is(err, workflow.ErrExportWouldOverwrite) {
			h.presenter.Error("%v", err)
			h.presenter.Info("Re-run with --force to replace them, or choose another --out directory.")
			return cli.Exit("", 1)
		}
		h.presenter.Error("Failed to export the run: %v", err)
		return err
	}

	h.presenter.Success("Exported %s to %s", id, out.Dir)
	table := h.presenter.NewTable([]string{"File", "Source", "Size", "Note"})
	for _, f := range out.Files {
		_ = table.Append([]string{f.Name, sourceLabel(f.Source), workflow.FormatBytes(f.Size), f.Warning})
	}
	_ = table.Append([]string{workflow.ExportManifestName, "pumbaa", "", "where each file came from"})
	_ = table.Render()

	h.presenter.Newline()
	for _, m := range out.Missing {
		h.presenter.Warning("No %s: %s.", m.Kind, m.Reason)
	}
	if len(out.Secrets) > 0 && !out.SecretsStripped {
		h.presenter.Warning("options.json carries credentials (%s). Re-run with --strip-secrets before handing the directory to someone else.", strings.Join(out.Secrets, ", "))
	}
	if out.Complete() {
		h.presenter.Info("Submit it again with:")
	} else {
		h.presenter.Info("Once the missing file is in place, submit it again with:")
	}
	h.presenter.Print("  cd %s && %s\n", out.Dir, out.SubmitCommand)
	if !out.Complete() {
		return cli.Exit("", 1)
	}
	return nil
}

func sourceLabel(source workflow.ExportSource) string {
	switch source {
	case workflow.ExportFromHistory:
		return "kept at submit"
	case workflow.ExportFromServer:
		return "server"
	case workflow.ExportFromPath:
		return "recorded path"
	}
	return string(source)
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
		h.presenter.Warning("This deletes every local record of runs submitted before %s, kept files included.", cutoff.Local().Format("2006-01-02"))
		h.presenter.Info("Re-run with --yes to confirm.")
		return nil
	}

	removed, err := h.useCase.Prune(c.Context, cutoff)
	if err != nil {
		h.presenter.Error("Failed to prune the local history: %v", err)
		return err
	}
	freed := ""
	if removed.Bytes > 0 {
		freed = fmt.Sprintf(", freeing %s of kept files", workflow.FormatBytes(removed.Bytes))
	}
	h.presenter.Success("Forgot %d run(s) submitted before %s%s.", removed.Runs, cutoff.Local().Format("2006-01-02"), freed)
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
