package handler

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// CacheForecastHandler handles the pre-submission cache prediction command.
type CacheForecastHandler struct {
	useCase   *workflow.CacheForecastUseCase
	presenter *presenter.Presenter
}

// NewCacheForecastHandler creates a new CacheForecastHandler.
func NewCacheForecastHandler(uc *workflow.CacheForecastUseCase, p *presenter.Presenter) *CacheForecastHandler {
	return &CacheForecastHandler{useCase: uc, presenter: p}
}

// Command returns the CLI command for the cache forecast.
func (h *CacheForecastHandler) Command() *cli.Command {
	return &cli.Command{
		Name:    "cache-forecast",
		Aliases: []string{"forecast"},
		Usage:   "Predict which tasks would be served from cache before submitting",
		Description: "Compares a pending submission against a previous run and reports which calls\n" +
			"would be reused and which would run again, so you can decide whether a run is\n" +
			"worth starting.\n\n" +
			"The prediction is advisory: Cromwell decides. Supported backends are local and\n" +
			"GCP; anything else is reported as undetermined rather than guessed.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workflow",
				Aliases:  []string{"w"},
				Usage:    "[required] Path to the WDL workflow file",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "inputs",
				Aliases: []string{"i"},
				Usage:   "[optional] Path to the inputs JSON file",
			},
			&cli.StringFlag{
				Name:    "against",
				Aliases: []string{"a"},
				Usage:   "[optional] Reference run ID; defaults to the latest successful run",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "[optional] Emit the forecast as JSON",
			},
		},
		Action: h.handle,
	}
}

func (h *CacheForecastHandler) handle(c *cli.Context) error {
	forecast, err := h.useCase.Execute(context.Background(), workflow.CacheForecastInput{
		WorkflowFile: c.String("workflow"),
		InputsFile:   c.String("inputs"),
		ReferenceID:  c.String("against"),
	})
	if err != nil {
		return err
	}

	if c.Bool("json") {
		return json.NewEncoder(os.Stdout).Encode(forecastJSON(forecast))
	}
	renderCacheForecast(h.presenter, forecast)
	return nil
}

// renderCacheForecast prints the headline first — how much of the run is free —
// then the root causes, which are the only thing the user can act on.
func renderCacheForecast(p *presenter.Presenter, f *domain.CacheForecast) {
	p.Title("Cache forecast")
	p.KeyValue("Reference run", f.Reference)
	p.KeyValue("Backend", f.Backend.String())
	p.Newline()

	counts := f.Counts()
	total := len(f.Calls)
	reuse := counts[domain.FateReuse]

	switch {
	case total == 0:
		p.Warning("no calls found in this workflow")
		return
	case reuse == total:
		p.Success("all %d call(s) would be served from cache — this run should cost nothing", total)
	case reuse == 0:
		p.Warning("none of the %d call(s) would be reused", total)
	default:
		p.Info("%d of %d call(s) would be served from cache", reuse, total)
	}

	if n := counts[domain.FateRerun]; n > 0 {
		p.Newline()
		p.Print("  Will run again (%d):\n", n)
		for _, c := range f.RootCauses() {
			p.Print("    ✗ %-24s %s\n", c.Call, strings.Join(c.Reasons, "; "))
		}
	}

	if n := counts[domain.FateRerunDownstream]; n > 0 {
		p.Newline()
		p.Print("  Downstream of those (%d) — likely to run again, not certain:\n", n)
		for _, c := range f.Calls {
			if c.Fate == domain.FateRerunDownstream {
				p.Print("    ↓ %-24s after %s\n", c.Call, c.Cause)
			}
		}
	}

	if n := counts[domain.FateUnknown]; n > 0 {
		p.Newline()
		p.Print("  Could not determine (%d):\n", n)
		for _, c := range f.Calls {
			if c.Fate == domain.FateUnknown {
				p.Print("    ? %-24s %s\n", c.Call, strings.Join(c.Reasons, "; "))
			}
		}
	}

	if len(f.Warnings) > 0 {
		p.Newline()
		for _, w := range f.Warnings {
			p.Warning("%s", w)
		}
	}

	p.Newline()
	p.Print("  %s\n", forecastCaveat(counts))
}

// forecastCaveat states the limit that applies to what was actually shown, so
// the caveat is never boilerplate the user learns to skip.
func forecastCaveat(counts map[domain.PredictedFate]int) string {
	if counts[domain.FateRerunDownstream] > 0 {
		return "Downstream calls are a prediction: a rerun task that produces byte-identical " +
			"outputs still hits the cache. Cromwell decides."
	}
	return "Advisory only — Cromwell decides."
}

// forecastJSON is the machine-readable shape, kept explicit so the CLI contract
// does not drift with the domain's internals.
func forecastJSON(f *domain.CacheForecast) map[string]any {
	counts := f.Counts()
	calls := make([]map[string]any, 0, len(f.Calls))
	for _, c := range f.Calls {
		entry := map[string]any{
			"call": c.Call,
			"fate": fateSlug(c.Fate),
		}
		if len(c.Reasons) > 0 {
			entry["reasons"] = c.Reasons
		}
		if c.Cause != "" {
			entry["cause"] = c.Cause
		}
		calls = append(calls, entry)
	}
	return map[string]any{
		"reference": f.Reference,
		"backend":   f.Backend.String(),
		"summary": map[string]int{
			"total":           len(f.Calls),
			"reuse":           counts[domain.FateReuse],
			"rerun":           counts[domain.FateRerun],
			"rerunDownstream": counts[domain.FateRerunDownstream],
			"undetermined":    counts[domain.FateUnknown],
		},
		"calls":    calls,
		"warnings": f.Warnings,
	}
}

func fateSlug(f domain.PredictedFate) string {
	switch f {
	case domain.FateReuse:
		return "reuse"
	case domain.FateRerun:
		return "rerun"
	case domain.FateRerunDownstream:
		return "rerun_downstream"
	default:
		return "undetermined"
	}
}
