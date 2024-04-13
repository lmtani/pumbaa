package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/pumbaa/internal/core/cromwell"
	"github.com/lmtani/pumbaa/internal/core/interactive"
	"github.com/lmtani/pumbaa/internal/core/local"

	"github.com/lmtani/pumbaa/internal/types"

	"github.com/lmtani/pumbaa/internal/adapters"
	urfaveCli "github.com/urfave/cli/v2"
)

func DefaultCromwell(h, iap string) *cromwell.Cromwell {
	gcp := adapters.NewGoogleCloud(iap)
	client := adapters.NewCromwellClient(h, gcp)
	return cromwell.NewCromwell(client, adapters.NewColoredWriter(os.Stdout))
}

func build(c *urfaveCli.Context) error {
	wdl := adapters.RegexWdlPArser{}
	fs := adapters.NewLocalFilesystem()
	releaser := local.NewBuilder(&wdl, fs)
	err := releaser.WorkflowDist(c.String("wdl"), c.String("out"))
	return err
}

func getVersion(b *Build) error {
	fmt.Printf("Version: %s\n", b.Version)
	fmt.Printf("Date: %s\n", b.Date)
	fmt.Printf("Commit: %s\n", b.Commit)
	return nil
}

func query(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	return cs.QueryWorkflow(c.String("name"), time.Duration(c.Int64("days")))
}

func wait(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	return cs.Wait(c.String("operation"), c.Int("sleep"))
}

func submit(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	return cs.SubmitWorkflow(c.String("wdl"), c.String("inputs"), c.String("dependencies"), c.String("options"))
}

func inputs(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	_, err := cs.Inputs(c.String("operation"))
	return err
}

func kill(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	_, err := cs.Kill(c.String("operation"))
	return err
}

func metadata(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	return cs.Metadata(c.String("operation"))
}

func outputs(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	return cs.Outputs(c.String("operation"))
}

func navigate(c *urfaveCli.Context) error {
	gcp := adapters.NewGoogleCloud(c.String("iap"))
	cc := adapters.NewCromwellClient(c.String("host"), gcp)
	w := adapters.NewColoredWriter(os.Stdout)
	ui := adapters.Ui{}
	n := interactive.NewNavigate(cc, w, &ui)
	return n.Navigate(c.String("operation"))
}

func gcpResources(c *urfaveCli.Context) error {
	cs := DefaultCromwell(c.String("host"), c.String("iap"))
	return cs.Get(c.String("operation"))
}

func localDeploy(c *urfaveCli.Context) error {
	config := ParseCliParams(c)
	db := adapters.NewMysql(config.Database)
	gcs := adapters.NewGoogleCloud("")
	fs := adapters.NewLocalFilesystem()
	h := adapters.NewDefaultHTTP()
	ld := local.NewDeployer(fs, db, gcs, h, config)
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
