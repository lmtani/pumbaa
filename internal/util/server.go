package util

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/schollz/progressbar/v3"
)

const CromwellUrl = "https://github.com/broadinstitute/cromwell/releases/download/85/cromwell-85.jar"

func StartCromwellServer(c Config, replaceConfig bool) error {
	err := checkRequirements(c.Database)
	if err != nil {
		return err
	}

	cromwell, err := cromwellSavePath()
	if err != nil {
		return err
	}

	// Download Cromwell if it does not exist
	_, err = os.Stat(cromwell)
	if os.IsNotExist(err) {
		err = DownloadCromwell(cromwell)
		if err != nil {
			return err
		}
	}

	basePath := filepath.Dir(cromwell)
	config := filepath.Join(basePath, "cromwell.conf")
	_, err = os.Stat(config)
	if os.IsNotExist(err) || replaceConfig {
		err = createCromwellConfig(config, c)
		if err != nil {
			return err
		}
	}

	err = startCromwellProcess(cromwell, config, basePath)
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
	fmt.Printf("cd %s && java -DLOG_MODE=pretty -Dconfig.file=%s -jar %s server", basePath, configFile, cromwellPath)
	return nil
}

func isInUserPath(s string) bool {
	_, err := exec.LookPath(s)
	return err == nil
}

func DownloadCromwell(cromwellFileName string) error {
	// create http client
	client := http.Client{
		Timeout: 60 * time.Second,
	}

	// get the content length of the file
	resp, err := client.Head(CromwellUrl)
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
	resp, err = client.Get(CromwellUrl)
	if err != nil {
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

func cromwellSavePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(home, ".cromwell")
	err = createDirectory(saveDir)
	if err != nil {
		return "", err
	}

	fileName := filepath.Join(saveDir, "cromwell.jar")
	return fileName, nil
}

func createDirectory(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		err := os.MkdirAll(p, 0777)
		if err != nil {
			return err
		}
	}
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
