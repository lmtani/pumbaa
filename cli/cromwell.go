package app

import (
	"github.com/lmtani/cromwell-cli/internal/util"
	"github.com/urfave/cli/v2"
)

func localDeploy() *cli.Command {
	return &cli.Command{
		Name:    "local-deploy",
		Aliases: []string{"ld"},
		Usage:   "Install Cromwell Server locally with default configuration and start it",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "mysql-host", Required: false, Value: "127.0.0.1", Usage: "Your MySQL host"},
			&cli.StringFlag{Name: "mysql-passwd", Aliases: []string{"d"}, Required: false, Value: "1234", Usage: "Your MySQL password"},
			&cli.StringFlag{Name: "mysql-user", Required: false, Value: "root", Usage: "Your MySQL user"},
			&cli.Int64Flag{Name: "mysql-port", Required: false, Value: 3306, Usage: "Your MySQL port"},
			&cli.Int64Flag{Name: "port", Required: false, Value: 8000, Usage: "Port to bind Cromwell Server"},
			&cli.Int64Flag{Name: "max-jobs", Required: false, Value: 1, Usage: "Maximum number of jobs to run in parallel"},
			&cli.Int64Flag{Name: "replace-config", Required: false, Value: 1, Usage: "Maximum number of jobs to run in parallel"},
			&cli.BoolFlag{Name: "override", Required: false, Usage: "Override the existing configuration file"},
		},
		Action: func(c *cli.Context) error {
			config := util.Config{
				Backend: util.Backend{
					Default:                "Local",
					Provider:               "Local",
					ActorFactory:           "cromwell.backend.impl.sfs.config.ConfigBackendLifecycleActorFactory",
					MaxConcurrentWorkflows: 100,
					ConcurrentJobLimit:     100,
				},
				Database: util.Database{
					Profile:           "slick.jdbc.MySQLProfile$",
					Driver:            "com.mysql.cj.jdbc.Driver",
					Host:              c.String("mysql-host"),
					User:              "root",
					Password:          "1234",
					Port:              3306,
					ConnectionTimeout: 50000,
				},
				CallCaching: util.CallCaching{
					Enabled:                   true,
					InvalidateBadCacheResults: true,
				},
				Docker: util.Docker{
					PerformRegistryLookupIfDigestIsProvided: false,
				},
			}
			return util.StartCromwellServer(config, c.Bool("override"))
		},
	}
}

func packDependencies() *cli.Command {
	return &cli.Command{
		Name:    "build",
		Aliases: []string{"b"},
		Usage:   "Edit import statements in WDLs and build a zip file with all dependencies",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "wdl", Required: true, Usage: "Main workflow"},
		},
		Action: func(c *cli.Context) error {
			return util.BuildWorkflowDist(c.String("wdl"))
		},
	}
}
