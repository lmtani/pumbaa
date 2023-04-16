package cromwell

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"
)

//go:embed config.tmpl
var ConfigTmpl string

func createCromwellConfig(savePath string, config Config) error {
	// Parse the template

	tmpl, err := template.New("config").Parse(ConfigTmpl)
	if err != nil {
		return err
	}

	// create a new file
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Render the template with the configuration values
	config.Database.URL = fmt.Sprintf(
		"jdbc:mysql://%s:%d/cromwell?rewriteBatchedStatements=true",
		config.Database.Host, config.Database.Port)
	err = tmpl.Execute(file, config)
	if err != nil {
		return err
	}
	return nil
}

type BackendConfig struct {
	Default   string
	Providers []ProviderConfig
}

type ProviderConfig struct {
	Name        string
	ActorFactor string
	Config      ProviderSettings
}

type ProviderSettings struct {
	MaxConcurrentWorkflows int
	ConcurrentJobLimit     int
	FileSystems            Engine
}

type GcsFilesystem struct {
	Auth string `json:"auth"`
}

type LocalFilesystem struct {
	Localization []string `json:"localization"`
}

type Filesystems struct {
	GcsFilesystem   `json:"gcs,omitempty"`
	HTTP            struct{} `json:"http,omitempty"`
	LocalFilesystem `json:"local,omitempty"`
}

type Engine struct {
	Filesystems `json:"filesystems"`
}

type Database struct {
	Profile           string
	Driver            string
	URL               string
	Host              string
	Port              int
	User              string
	Password          string
	ConnectionTimeout int
}

type CallCaching struct {
	Enabled                   bool
	InvalidateBadCacheResults bool
}

type Docker struct {
	PerformRegistryLookupIfDigestIsProvided bool
}

type Config struct {
	BackendConfig
	Database
	CallCaching
	Docker
	Engine
}

func ParseCliParams(c *cli.Context) Config {
	engines := Engine{
		Filesystems{
			HTTP:            struct{}{},
			GcsFilesystem:   GcsFilesystem{Auth: "application-default"},
			LocalFilesystem: LocalFilesystem{Localization: []string{"hard-link", "soft-link", "copy"}},
		},
	}

	config := Config{
		BackendConfig: BackendConfig{
			Default: "Local",
			Providers: []ProviderConfig{
				{Name: "Local", ActorFactor: "cromwell.backend.impl.sfs.config.ConfigBackendLifecycleActorFactory", Config: ProviderSettings{MaxConcurrentWorkflows: 1, ConcurrentJobLimit: c.Int("max-jobs"), FileSystems: engines}},
			},
		},
		Database: Database{
			Profile:           "slick.jdbc.MySQLProfile$",
			Driver:            "com.mysql.cj.jdbc.Driver",
			Host:              c.String("mysql-host"),
			User:              c.String("mysql-user"),
			Password:          c.String("mysql-passwd"),
			Port:              c.Int("mysql-port"),
			ConnectionTimeout: 50000,
		},
		CallCaching: CallCaching{
			Enabled:                   true,
			InvalidateBadCacheResults: true,
		},
		Docker: Docker{
			PerformRegistryLookupIfDigestIsProvided: false,
		},
		Engine: engines,
	}
	return config
}
