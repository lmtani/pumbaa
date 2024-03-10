package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/core"
	urfaveCli "github.com/urfave/cli/v2"
)

func build(c *urfaveCli.Context) error {
	wdl := adapters.RegexWdlPArser{}
	fs := adapters.LocalFilesystem{}
	releaser := core.NewRelease(&wdl, &fs)
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
	return i.Inputs(c.String("operation"))
}

func kill(c *urfaveCli.Context) error {
	cromwellClient := adapters.NewCromwellClient(c.String("host"), c.String("iap"))
	k := core.NewKill(cromwellClient)
	return k.Kill(c.String("operation"))
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
