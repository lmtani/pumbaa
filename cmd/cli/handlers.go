package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/pumbaa/internal/types"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/core"
	urfaveCli "github.com/urfave/cli/v2"
)

func build(c *urfaveCli.Context) error {
	wdl := adapters.RegexWdlPArser{}
	fs := adapters.NewLocalFilesystem()
	releaser := core.NewRelease(&wdl, fs)
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
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	writer := adapters.NewColoredWriter(os.Stdout)
	q := core.NewQuery(cromwellClient, writer)
	return q.QueryWorkflow(c.String("name"), time.Duration(c.Int64("days")))
}

func wait(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	writer := adapters.NewColoredWriter(os.Stdout)
	w := core.NewWait(cromwellClient, writer)
	return w.Wait(c.String("operation"), c.Int("sleep"))
}

func submit(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	writer := adapters.NewColoredWriter(os.Stdout)
	s := core.NewSubmit(cromwellClient, writer)
	return s.SubmitWorkflow(c.String("wdl"), c.String("inputs"), c.String("dependencies"), c.String("options"))
}

func inputs(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	i := core.NewInputs(cromwellClient)
	_, err := i.Inputs(c.String("operation"))
	return err
}

func kill(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	w := adapters.NewColoredWriter(os.Stdout)
	k := core.NewKill(cromwellClient, w)
	_, err := k.Kill(c.String("operation"))
	return err
}

func metadata(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	writer := adapters.NewColoredWriter(os.Stdout)
	m := core.NewMetadata(cromwellClient, writer)
	return m.Metadata(c.String("operation"))
}

func outputs(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	out := core.NewOutputs(cromwellClient)
	return out.Outputs(c.String("operation"))
}

func navigate(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	writer := adapters.NewColoredWriter(os.Stdout)
	ui := adapters.Ui{}
	n := core.NewNavigate(cromwellClient, writer, &ui)
	return n.Navigate(c.String("operation"))
}

func gcpResources(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	writer := adapters.NewColoredWriter(os.Stdout)
	resources := core.NewResourcesUsed(cromwellClient, writer)
	return resources.Get(c.String("operation"))
}

func localDeploy(c *urfaveCli.Context) error {
	config := ParseCliParams(c)
	mysql := adapters.NewMysql(config.Database)
	gcs := adapters.NewGoogleStorage()
	fs := adapters.NewLocalFilesystem()
	h := adapters.NewDefaultHTTP()
	ld := core.NewLocalDeploy(fs, mysql, gcs, h, config)
	return ld.Deploy()
}

// ParseCliParams Auxiliar for local deployment
func ParseCliParams(c *urfaveCli.Context) types.Config {
	engines := types.Engine{
		Filesystems: types.Filesystems{
			HTTP:            struct{}{},
			GcsFilesystem:   types.GcsFilesystem{Auth: "application-default", Enabled: true},
			LocalFilesystem: types.LocalFilesystem{Localization: []string{"hard-link", "soft-link", "copy"}},
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
