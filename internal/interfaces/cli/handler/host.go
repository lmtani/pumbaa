package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// HealthCheckerFactory builds a health checker aimed at an arbitrary URL, so
// `host check` can reach servers other than the active one without the
// interface layer knowing how a Cromwell client is built.
type HealthCheckerFactory func(url string) ports.HealthChecker

// HostHandler manages the registry of Cromwell hosts.
type HostHandler struct {
	presenter *presenter.Presenter
	// active is the host this invocation resolved to, shown in listings so
	// "which server am I talking to" is answerable without guessing.
	active   func() config.HostRef
	newCheck HealthCheckerFactory
}

// NewHostHandler creates a new HostHandler.
func NewHostHandler(p *presenter.Presenter, active func() config.HostRef, newCheck HealthCheckerFactory) *HostHandler {
	return &HostHandler{presenter: p, active: active, newCheck: newCheck}
}

// Command returns the CLI command for host management.
func (h *HostHandler) Command() *cli.Command {
	return &cli.Command{
		Name:  "host",
		Usage: "Manage Cromwell server aliases",
		Description: `Registers Cromwell servers under short names, so they can be selected
by alias instead of URL: pumbaa --host prod workflow submit ...

The alias also works in CROMWELL_HOST. With no host given, the default
alias is used.`,
		Action: func(c *cli.Context) error { return h.list() },
		Subcommands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Register a Cromwell server under an alias",
				ArgsUsage: "<alias> <url>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "default", Usage: "[optional] Also make it the default host"},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() < 2 {
						return fmt.Errorf("usage: pumbaa host add <alias> <url>")
					}
					return h.add(c.Args().Get(0), c.Args().Get(1), c.Bool("default"))
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List registered hosts",
				Action:  func(c *cli.Context) error { return h.list() },
			},
			{
				Name:      "use",
				Usage:     "Make a registered host the default",
				ArgsUsage: "<alias>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						return fmt.Errorf("usage: pumbaa host use <alias>")
					}
					return h.use(c.Args().Get(0))
				},
			},
			{
				Name:      "remove",
				Aliases:   []string{"rm"},
				Usage:     "Remove a registered host",
				ArgsUsage: "<alias>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						return fmt.Errorf("usage: pumbaa host remove <alias>")
					}
					return h.remove(c.Args().Get(0))
				},
			},
			{
				Name:      "check",
				Usage:     "Check whether hosts are reachable",
				ArgsUsage: "[alias]",
				Action: func(c *cli.Context) error {
					return h.check(c.Context, c.Args().First())
				},
			},
		},
	}
}

func (h *HostHandler) add(alias, url string, makeDefault bool) error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}
	if err := cfg.AddHost(alias, url); err != nil {
		return err
	}
	if makeDefault {
		if err := cfg.SetDefaultHost(alias); err != nil {
			return err
		}
	}
	if err := config.SaveFileConfig(cfg); err != nil {
		return err
	}

	h.presenter.Success("Host %q registered: %s", alias, cfg.Hosts[alias])
	if cfg.DefaultHost == alias {
		h.presenter.Info("It is now the default host.")
	} else {
		h.presenter.Info("Use it with: pumbaa --host %s <command>", alias)
	}
	return nil
}

func (h *HostHandler) list() error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}

	active := h.active()
	if len(cfg.Hosts) == 0 {
		h.presenter.Info("No hosts registered.")
		h.presenter.KeyValue("Current host", active.URL)
		h.presenter.Newline()
		h.presenter.Info("Register one with: pumbaa host add local %s", active.URL)
		return nil
	}

	h.presenter.Title("Cromwell hosts")
	table := h.presenter.NewTable([]string{"", "Alias", "URL"})
	for _, alias := range cfg.HostAliases() {
		_ = table.Append([]string{hostMarker(alias, cfg.DefaultHost, active), alias, cfg.Hosts[alias]})
	}
	_ = table.Render()

	h.presenter.Newline()
	h.presenter.Info("→ in use now   * default")
	if active.Alias == "" {
		h.presenter.Info("Current host is not registered: %s", active.URL)
	}
	return nil
}

// hostMarker distinguishes the host in use from the configured default: they
// differ whenever --host or CROMWELL_HOST overrode the default, and that is
// exactly when a reader needs to be told.
func hostMarker(alias, defaultAlias string, active config.HostRef) string {
	switch {
	case alias == active.Alias && alias == defaultAlias:
		return "→*"
	case alias == active.Alias:
		return "→"
	case alias == defaultAlias:
		return "*"
	default:
		return ""
	}
}

func (h *HostHandler) use(alias string) error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}
	if err := cfg.SetDefaultHost(alias); err != nil {
		return err
	}
	if err := config.SaveFileConfig(cfg); err != nil {
		return err
	}
	h.presenter.Success("Default host is now %q (%s)", alias, cfg.Hosts[alias])
	return nil
}

func (h *HostHandler) remove(alias string) error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}
	if err := cfg.RemoveHost(alias); err != nil {
		return err
	}
	if err := config.SaveFileConfig(cfg); err != nil {
		return err
	}
	h.presenter.Success("Host %q removed.", alias)
	if cfg.DefaultHost != "" {
		h.presenter.Info("Default host is now %q.", cfg.DefaultHost)
	}
	return nil
}

// check reports reachability. A host that cannot be reached is a normal
// answer here, not a failure of the command: the whole point is to find out.
func (h *HostHandler) check(ctx context.Context, alias string) error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return err
	}

	targets := []config.HostRef{}
	if alias != "" {
		ref, err := cfg.ResolveHost(alias)
		if err != nil {
			return err
		}
		targets = append(targets, ref)
	} else if len(cfg.Hosts) == 0 {
		targets = append(targets, h.active())
	} else {
		for _, a := range cfg.HostAliases() {
			targets = append(targets, config.HostRef{Alias: a, URL: cfg.Hosts[a]})
		}
	}

	table := h.presenter.NewTable([]string{"Host", "URL", "Status", "Latency"})
	for _, target := range targets {
		status, latency := h.probe(ctx, target.URL)
		_ = table.Append([]string{target.Display(), target.URL, status, latency})
	}
	_ = table.Render()
	return nil
}

func (h *HostHandler) probe(ctx context.Context, url string) (status, latency string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	health, err := h.newCheck(url).GetHealthStatus(ctx)
	elapsed := time.Since(start).Round(time.Millisecond)
	switch {
	case err != nil:
		return "unreachable", "-"
	case !health.OK:
		return fmt.Sprintf("degraded (%v)", health.UnhealthySystems), elapsed.String()
	default:
		return "ok", elapsed.String()
	}
}
