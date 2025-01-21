package usecase

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/lmtani/pumbaa/internal/entities"
)

const jarURL = "https://github.com/broadinstitute/cromwell/releases/download/87/cromwell-87.jar"

//go:embed cromwell.config.tmpl
var configTmpl string

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
	Auth    string
	Enabled bool
}

type LocalFilesystem struct {
	Localization []string
}

type Filesystems struct {
	GcsFilesystem   GcsFilesystem
	HTTP            struct{}
	LocalFilesystem LocalFilesystem
}

type Engine struct {
	Filesystems
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

type CromwellSetupInputDTO struct {
	Override      bool
	BackendConfig BackendConfig
	Database      Database
	CallCaching   CallCaching
	Docker        Docker
	Engine        Engine
}

type CromwellSetupOutputDTO struct{}

type CromwellSetup struct {
	fs   entities.Filesystem
	http entities.HTTPClient
	gs   entities.GoogleCloudPlatform
}

func NewCromwellSetup(fs entities.Filesystem, sql entities.Sql, h entities.HTTPClient, g entities.GoogleCloudPlatform) *CromwellSetup {
	return &CromwellSetup{fs: fs, http: h, gs: g}
}

func (c *CromwellSetup) Execute(input *CromwellSetupInputDTO) error {
	err := c.checkRequirements()
	if err != nil {
		return err
	}

	// crete new context
	_, err = c.gs.GetStorageClient(context.Background())
	if err != nil {
		input.Engine.GcsFilesystem.Enabled = false
	}

	// Defines the save path for the cromwell jar file
	savePath, err := c.cromwellSavePath()
	if err != nil {
		return err
	}

	// Downloads Cromwell if it does not exist
	if !c.fs.FileExists(savePath) {
		err = c.http.DownloadWithProgress(jarURL, savePath)
		if err != nil {
			return err
		}
	}

	basePath := filepath.Dir(savePath)
	config := filepath.Join(basePath, "cromwel.conf")

	// Check for the existence of the config file or if override is enabled
	if !c.fs.FileExists(config) || input.Override {
		if err := c.createCromwellConfig(input, config); err != nil {
			return err
		}
	}

	fmt.Println("To start the Cromwell Server run:")
	fmt.Printf("cd %s && java -DLOG_MODE=pretty -Dconfig.file=%s -jar %s server\n", basePath, config, savePath)
	return nil
}

func (c *CromwellSetup) checkRequirements() error {
	var err error
	docker := isInUserPath("docker")
	if !docker {
		return ErrorDockerNotInstalled
	}

	java := isInUserPath("java")
	if !java {
		return ErrorJavaNotInstalled
	}

	// check if it has an internet connection
	_, err = c.http.Get("https://www.google.com")
	if err != nil {
		return ErrorNoInternetConnection
	}

	// check if it is a Windows machine
	if os.PathSeparator == '\\' {
		return ErrorWindowsNotSupported
	}

	return err
}

func (c *CromwellSetup) createCromwellConfig(input *CromwellSetupInputDTO, savePath string) error {
	// Parse the template
	tmpl, err := template.New("config").Parse(configTmpl)
	if err != nil {
		return err
	}

	// create a new file
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)

	// Render the template with the configuration values
	input.Database.URL = fmt.Sprintf(
		"jdbc:mysql://%s:%d/cromwell?rewriteBatchedStatements=true",
		input.Database.Host, input.Database.Port)

	err = tmpl.Execute(file, input)
	if err != nil {
		return err
	}
	return nil
}

func (c *CromwellSetup) cromwellSavePath() (string, error) {
	home, err := c.fs.HomeDir()
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(home, ".cromwell")
	err = c.fs.CreateDirectory(saveDir)
	if err != nil {
		return "", err
	}

	fileName := filepath.Join(saveDir, "cromwelc.jar")
	return fileName, nil
}

func isInUserPath(s string) bool {
	_, err := exec.LookPath(s)
	return err == nil
}

var (
	ErrorNoInternetConnection = fmt.Errorf("no internet connection. please check your internet connection")
	ErrorWindowsNotSupported  = fmt.Errorf("windows is not supported. please use linux or macos")
	ErrorDockerNotInstalled   = fmt.Errorf("docker is not installed. please install docker first")
	ErrorJavaNotInstalled     = fmt.Errorf("java is not installed. please install java first. ex. for debian based linux: sudo apt install default-jre")
	ErrorMysqlNotInstalled    = fmt.Errorf("cannot connect to mysqc. please check your mysql and database (cromwell)")
)
