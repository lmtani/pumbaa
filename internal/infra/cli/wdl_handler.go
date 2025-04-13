package cli

import (
	"github.com/lmtani/pumbaa/internal/interfaces"
	"github.com/lmtani/pumbaa/internal/usecase"
	urfaveCli "github.com/urfave/cli/v2"
)

type WDLHandler struct {
	DB         interfaces.Sql
	FileSystem interfaces.Filesystem
	HTTP       interfaces.HTTPClient
	gcs        interfaces.GoogleCloudPlatform
	WDL        interfaces.Wdl
}

func NewWdlHandler(fs interfaces.Filesystem, http interfaces.HTTPClient, gcs interfaces.GoogleCloudPlatform, wdl interfaces.Wdl) *WDLHandler {
	return &WDLHandler{FileSystem: fs, HTTP: http, gcs: gcs, WDL: wdl}
}

func (h *WDLHandler) Deploy(c *urfaveCli.Context) error {
	input := ParseCliParams(c)
	cromwellSetup := usecase.NewCromwellSetup(h.FileSystem, h.DB, h.HTTP, h.gcs)
	return cromwellSetup.Execute(input)
}

func (h *WDLHandler) Build(c *urfaveCli.Context) error {
	input := usecase.BuildInputDTO{
		WDLPath: c.String("wdl"),
		OutPath: c.String("out"),
	}
	cromwellSetup := usecase.NewWDLBuilder(h.FileSystem, h.WDL)
	return cromwellSetup.Execute(input)
}

// ParseCliParams Auxiliar for local deployment
func ParseCliParams(c *urfaveCli.Context) *usecase.CromwellSetupInputDTO {
	engines := usecase.Engine{
		Filesystems: usecase.Filesystems{
			HTTP:            struct{}{},
			GcsFilesystem:   usecase.GcsFilesystem{Auth: "application-default", Enabled: true},
			LocalFilesystem: usecase.LocalFilesystem{Localization: []string{"hard-link", "soft-link", "cached-copy"}},
		},
	}

	config := usecase.CromwellSetupInputDTO{
		Override: c.Bool("override"),
		BackendConfig: usecase.BackendConfig{
			Default: "Local",
			Providers: []usecase.ProviderConfig{
				{
					Name:        "Local",
					ActorFactor: "cromwell.backend.impl.sfs.config.ConfigBackendLifecycleActorFactory",
					Config: usecase.ProviderSettings{
						MaxConcurrentWorkflows: 1,
						ConcurrentJobLimit:     c.Int("max-jobs"),
						FileSystems:            engines,
					},
				},
			},
		},
		Database: usecase.Database{
			Profile:           "slick.jdbc.MySQLProfile$",
			Driver:            "com.mysql.cj.jdbc.Driver",
			Host:              c.String("mysql-host"),
			User:              c.String("mysql-user"),
			Password:          c.String("mysql-passwd"),
			Port:              c.Int("mysql-port"),
			ConnectionTimeout: 50000,
		},
		CallCaching: usecase.CallCaching{
			Enabled:                   true,
			InvalidateBadCacheResults: true,
		},
		Docker: usecase.Docker{
			PerformRegistryLookupIfDigestIsProvided: false,
		},
		Engine: engines,
	}
	return &config
}
