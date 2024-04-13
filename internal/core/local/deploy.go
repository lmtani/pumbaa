package local

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/lmtani/pumbaa/internal/ports"
	"github.com/lmtani/pumbaa/internal/types"
)

const jarUrl = "https://github.com/broadinstitute/cromwell/releases/download/85/cromwell-85.jar"

//go:embed config.tmpl
var ConfigTmpl string

type Deployer struct {
	fl   ports.Filesystem
	http ports.HTTPClient
	sql  ports.Sql
	gs   ports.GoogleCloudPlatform
	c    types.Config
}

func NewDeployer(fl ports.Filesystem, sql ports.Sql, gs ports.GoogleCloudPlatform, h ports.HTTPClient, c types.Config) *Deployer {
	return &Deployer{fl: fl, sql: sql, c: c, gs: gs, http: h}
}

func (l *Deployer) Deploy() error {
	err := l.checkRequirements()
	if err != nil {
		return err
	}

	_, err = l.gs.GetStorageClient()
	if err != nil {
		fmt.Println(ErrorGoogleCredentials)
		l.c.Engine.GcsFilesystem.Enabled = false
	}

	// Defines the save path for the cromwell jar file
	savePath, err := l.CromwellSavePath()
	if err != nil {
		return err
	}

	// Downloads Cromwell if it does not exist
	_, err = os.Stat(savePath)
	if os.IsNotExist(err) {
		err = l.http.DownloadWithProgress(jarUrl, savePath)
		if err != nil {
			err = os.Remove(savePath)
			return err
		}
	}

	basePath := filepath.Dir(savePath)
	config := filepath.Join(basePath, "cromwell.conf")

	// Check for the existence of the config file or if override is enabled
	if _, err := os.Stat(config); os.IsNotExist(err) || l.c.Override {
		if err := l.createCromwellConfig(config); err != nil {
			return err
		}
	}

	fmt.Println("To start the Cromwell Server run:")
	fmt.Printf("cd %s && java -DLOG_MODE=pretty -Dconfig.file=%s -jar %s server\n", basePath, config, basePath)
	return nil
}

func (l *Deployer) checkRequirements() error {
	var err error
	docker := isInUserPath("docker")
	if !docker {
		return ErrorDockerNotInstalled
	}

	java := isInUserPath("java")
	if !java {
		return ErrorJavaNotInstalled
	}

	err = l.sql.CheckConnection()
	if err != nil {
		fmt.Println(err)
		return ErrorMysqlNotInstalled
	}
	// check if it has an internet connection
	_, err = http.Get("https://www.google.com")
	if err != nil {
		return ErrorNoInternetConnection
	}

	// check if it is a Windows machine
	if os.PathSeparator == '\\' {
		return ErrorWindowsNotSupported
	}

	return err
}

func (l *Deployer) createCromwellConfig(savePath string) error {
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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)

	// Render the template with the configuration values
	l.c.Database.URL = fmt.Sprintf(
		"jdbc:mysql://%s:%d/cromwell?rewriteBatchedStatements=true",
		l.c.Database.Host, l.c.Database.Port)

	err = tmpl.Execute(file, l.c)
	if err != nil {
		return err
	}
	return nil
}

func (l *Deployer) CromwellSavePath() (string, error) {
	home, err := l.fl.HomeDir()
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(home, ".cromwell")
	err = l.fl.CreateDirectory(saveDir)
	if err != nil {
		return "", err
	}

	fileName := filepath.Join(saveDir, "cromwell.jar")
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
	ErrorGoogleCredentials    = fmt.Errorf("google Cloud Default credentials not found. Disabling GCS filesystem")
	ErrorMysqlNotInstalled    = fmt.Errorf(`cannot connect to mysql. please check your mysql and database (cromwell).

			Start a new mysql server with:
			  - docker run -d --env MYSQL_ROOT_PASSWORD=1234 --env MYSQL_DATABASE=cromwell --name cromwell-db -p 3306:3306 mysql:8.0
			Stop it later with:
			  - docker stop cromwell-db
			Start it again with:
			  - docker start cromwell-db
		`)
)
