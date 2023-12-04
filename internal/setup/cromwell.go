package setup

import (
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/lmtani/pumbaa/internal/pkg/storage/google"
	"github.com/lmtani/pumbaa/internal/pkg/util"

	_ "github.com/go-sql-driver/mysql"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

const jarUrl = "https://github.com/broadinstitute/cromwell/releases/download/85/cromwell-85.jar"

//go:embed config.tmpl
var ConfigTmpl string

func StartCromwellServer(c Config, replaceConfig bool) error {
	err := checkRequirements(c.Database)
	if err != nil {
		return err
	}

	// Defines the save path for the cromwell jar file
	jarPath, err := util.CromwellSavePath()
	if err != nil {
		return err
	}

	// Downloads Cromwell if it does not exist
	_, err = os.Stat(jarPath)
	if os.IsNotExist(err) {
		err = DownloadCromwell(jarPath)
		if err != nil {
			return err
		}
	}

	basePath := filepath.Dir(jarPath)
	config := filepath.Join(basePath, "cromwell.conf")
	_, err = os.Stat(config)
	if os.IsNotExist(err) || replaceConfig {
		err = createCromwellConfig(config, c)
		if err != nil {
			return err
		}
	}

	err = startCromwellProcess(jarPath, config, basePath)
	if err != nil {
		return err
	}
	return nil
}

func checkRequirements(db Database) error {
	var err error
	docker := isInUserPath("docker")
	if !docker {
		return fmt.Errorf("docker is not installed. please install docker first")
	}

	java := isInUserPath("java")
	if !java {
		return fmt.Errorf("java is not installed. please install java first. ex. for debian based linux: sudo apt install default-jre")
	}

	err = checkMysqlConn(db)
	if err != nil {
		return fmt.Errorf(`cannot connect to mysql. please check your mysql and database (cromwell).

			Start a new mysql server with:
			  - docker run -d --env MYSQL_ROOT_PASSWORD=1234 --env MYSQL_DATABASE=cromwell --name cromwell-db -p 3306:3306 mysql:8.0
			Stop it later with:
			  - docker stop cromwell-db
			Start it again with:
			  - docker start cromwell-db
		`)
	}
	// check if it has internet connection
	_, err = http.Get("https://www.google.com")
	if err != nil {
		return fmt.Errorf("no internet connection. please check your internet connection")
	}

	// check if it is a Windows machine
	if os.PathSeparator == '\\' {
		return fmt.Errorf("windows is not supported. please use linux or macos")
	}

	return err
}

func startCromwellProcess(cromwellPath, configFile, basePath string) error {
	fmt.Println("To start the Cromwell Server run:")
	fmt.Printf("cd %s && java -DLOG_MODE=pretty -Dconfig.file=%s -jar %s server\n", basePath, configFile, cromwellPath)
	return nil
}

func isInUserPath(s string) bool {
	_, err := exec.LookPath(s)
	return err == nil
}

func DownloadCromwell(cromwellFileName string) error {
	// create http client
	client := http.Client{
		Timeout: 5 * time.Minute,
	}

	// get the content length of the file
	resp, err := client.Head(jarUrl)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)
	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	// create the progress bar
	bar := progressbar.DefaultBytes(
		int64(size),
		"downloading",
	)

	// download the file and update the progress bar
	resp, err = client.Get(jarUrl)
	if err != nil {
		// if the download fails, delete the file
		err = os.Remove(cromwellFileName)
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	file, err := os.Create(cromwellFileName)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)

	writer := io.MultiWriter(file, bar)

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("\nFile downloaded successfully.")
	return nil
}

type MysqlConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

func checkMysqlConn(dbConf Database) error {
	dbConn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port)
	db, err := sql.Open("mysql", dbConn)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("Failed to close database connection:", err)
		}
	}(db)

	err = db.Ping()
	if err != nil {
		return err
	}
	return nil
}

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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)

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

func ParseCliParams(c *cli.Context) Config {
	engines := Engine{
		Filesystems{
			HTTP:            struct{}{},
			GcsFilesystem:   GcsFilesystem{Auth: "application-default", Enabled: true},
			LocalFilesystem: LocalFilesystem{Localization: []string{"hard-link", "soft-link", "copy"}},
		},
	}

	_, err := google.GetClient()
	if err != nil {
		fmt.Println("Google Cloud Default credentials not found. Disabling GCS filesystem.")
		engines.GcsFilesystem.Enabled = false
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
