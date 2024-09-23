package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/pumbaa/internal/adapters/cromwellclient"
	"github.com/lmtani/pumbaa/internal/adapters/filesystem"
	"github.com/lmtani/pumbaa/internal/adapters/gcp"
	"github.com/lmtani/pumbaa/internal/adapters/http"
	"github.com/lmtani/pumbaa/internal/adapters/logger"
	"github.com/lmtani/pumbaa/internal/adapters/mysql"
	"github.com/lmtani/pumbaa/internal/adapters/prompt"
	"github.com/lmtani/pumbaa/internal/adapters/wdl"
	"github.com/lmtani/pumbaa/internal/adapters/writer"

	"github.com/lmtani/pumbaa/internal/core/cromwell"
	"github.com/lmtani/pumbaa/internal/core/interactive"
	"github.com/lmtani/pumbaa/internal/core/local"
	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
	urfaveCli "github.com/urfave/cli/v2"
)

func DefaultCromwell(h, aud string) *cromwell.Cromwell {
	var googleCloud ports.GoogleCloudPlatform
	if aud != "" {
		googleCloud = DefaultGcp(aud)
	}

	client := cromwellclient.NewCromwellClient(h, googleCloud)
	return cromwell.NewCromwell(client, &logger.Logger{})
}

func DefaultGcp(aud string) ports.GoogleCloudPlatform {
	return gcp.NewGoogleCloud(aud, &gcp.Wrapper{})
}

type Handler struct {
	c *cromwell.Cromwell
	w ports.Writer
}

func NewDefaultHandler(h, iap string) *Handler {
	c := DefaultCromwell(h, iap)
	w := writer.NewColoredWriter(os.Stdout)
	return &Handler{c: c, w: w}
}

func (h *Handler) Query(c *urfaveCli.Context) error {
	d, err := h.c.QueryWorkflow(c.String("name"), time.Duration(c.Int64("days")))
	if err != nil {
		return err
	}
	h.w.QueryTable(d)
	return nil
}

func (h *Handler) wait(c *urfaveCli.Context) error {
	return h.c.Wait(c.String("operation"), c.Int("sleep"))
}

func (h *Handler) submit(c *urfaveCli.Context) error {
	d, err := h.c.SubmitWorkflow(c.String("wdl"), c.String("inputs"), c.String("dependencies"), c.String("options"))
	if err != nil {
		return err
	}
	return h.w.Json(d)
}

func (h *Handler) inputs(c *urfaveCli.Context) error {
	d, err := h.c.Inputs(c.String("operation"))
	if err != nil {
		return err
	}
	return h.w.Json(d)
}

func (h *Handler) kill(c *urfaveCli.Context) error {
	d, err := h.c.Kill(c.String("operation"))
	if err != nil {
		return err
	}
	return h.w.Json(d)
}

func (h *Handler) metadata(c *urfaveCli.Context) error {
	d, err := h.c.Metadata(c.String("operation"))
	if err != nil {
		return err
	}
	return h.w.MetadataTable(d)
}

func (h *Handler) outputs(c *urfaveCli.Context) error {
	d, err := h.c.Outputs(c.String("operation"))
	if err != nil {
		return err
	}
	return h.w.Json(d)
}

func (h *Handler) gcpResources(c *urfaveCli.Context) error {
	d, err := h.c.ResourceUsages(c.String("operation"))
	if err != nil {
		return err
	}
	h.w.ResourceTable(d)
	return nil
}

func packDependencies(c *urfaveCli.Context) error {
	parser := wdl.RegexWdlPArser{}
	l := logger.Logger{}
	fs := filesystem.NewLocalFilesystem(&l)
	releaser := local.NewBuilder(&parser, fs)
	err := releaser.PackDependencies(c.String("wdl"), c.String("out"))
	return err
}

func getVersion(b *Build) error {
	fmt.Printf("Version: %s\n", b.Version)
	fmt.Printf("Date: %s\n", b.Date)
	fmt.Printf("Commit: %s\n", b.Commit)
	return nil
}

func navigate(c *urfaveCli.Context) error {
	host, aud := c.String("host"), c.String("iap")
	var googleCloud ports.GoogleCloudPlatform
	if aud != "" {
		googleCloud = DefaultGcp(aud)
	}

	client := cromwellclient.NewCromwellClient(host, googleCloud)
	w := writer.NewColoredWriter(os.Stdout)
	ui := prompt.Ui{}
	n := interactive.NewNavigate(client, w, &ui)
	return n.Navigate(c.String("operation"))
}

func localDeploy(c *urfaveCli.Context) error {
	config := ParseCliParams(c)
	db := mysql.NewMysql(config.Database)
	googleClient := DefaultGcp("")
	l := logger.Logger{}
	fs := filesystem.NewLocalFilesystem(&l)
	h := http.NewDefaultHTTP()
	ld := local.NewDeployer(fs, db, googleClient, h, config)
	return ld.Deploy()
}

// ParseCliParams Auxiliar for local deployment
func ParseCliParams(c *urfaveCli.Context) types.Config {
	engines := types.Engine{
		Filesystems: types.Filesystems{
			HTTP:            struct{}{},
			GcsFilesystem:   types.GcsFilesystem{Auth: "application-default", Enabled: true},
			LocalFilesystem: types.LocalFilesystem{Localization: []string{"hard-link", "soft-link", "cached-copy"}},
		},
	}

	config := types.Config{
		Override: c.Bool("override"),
		BackendConfig: types.BackendConfig{
			Default: "Local",
			Providers: []types.ProviderConfig{
				{
					Name:        "Local",
					ActorFactor: "cromwell.backend.impl.sfs.config.ConfigBackendLifecycleActorFactory",
					Config: types.ProviderSettings{
						MaxConcurrentWorkflows: 1,
						ConcurrentJobLimit:     c.Int("max-jobs"),
						FileSystems:            engines,
					},
				},
			},
		},
		Database: types.Database{
			Profile:           "slick.jdbc.MySQLProfile$",
			Driver:            "com.mysql.cj.jdbc.Driver",
			Host:              c.String("mysql-host"),
			User:              c.String("mysql-user"),
			Password:          c.String("mysql-passwd"),
			Port:              c.Int("mysql-port"),
			ConnectionTimeout: 50000,
		},
		CallCaching: types.CallCaching{
			Enabled:                   true,
			InvalidateBadCacheResults: true,
		},
		Docker: types.Docker{
			PerformRegistryLookupIfDigestIsProvided: false,
		},
		Engine: engines,
	}
	return config
}
