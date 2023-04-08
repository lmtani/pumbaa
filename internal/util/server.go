package util

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"

	_ "github.com/go-sql-driver/mysql"
	"github.com/schollz/progressbar/v3"
)

const CromwellUrl = "https://github.com/broadinstitute/cromwell/releases/download/85/cromwell-85.jar"

func StartCromwellServer(db MysqlConfig, webport, maxJobs int, replaceConfig bool) error {

	// create a channel to receive signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	docker := isInUserPath("docker")
	if !docker {
		return fmt.Errorf("docker is not installed. please install docker first")
	}

	java := isInUserPath("java")
	if !java {
		return fmt.Errorf("java is not installed. please install java first. ex. for debian based linux: sudo apt install default-jre")
	}

	err := checkMysqlConn(db)
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

	cromwell, err := cromwellSavePath()
	if err != nil {
		return err
	}

	_, err = os.Stat(cromwell)
	if os.IsNotExist(err) {
		err = DownloadCromwell(cromwell)
		if err != nil {
			return err
		}
	}

	// get path before the last slash
	configPath := filepath.Dir(cromwell)

	config := filepath.Join(configPath, "cromwell.json")
	_, err = os.Stat(config)
	if os.IsNotExist(err) || replaceConfig {
		err = createDefaultConfig(db, config, webport, maxJobs)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("config file already exists. skipping...")
	}

	err = startCromwellProcess(cromwell, config)
	if err != nil {
		return err
	}
	return nil
}

func startCromwellProcess(cromwellPath, configFile string) error {
	cmd := exec.Command("java", "-DLOG_MODE=pretty", fmt.Sprintf("-Dconfig.file=%s", configFile), "-jar", cromwellPath, "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(cromwellPath)
	err := cmd.Start()
	if err != nil {
		return err
	}

	// Create a channel to receive signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a WaitGroup to wait for the goroutine to exit
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Start a goroutine to wait for signals and exit gracefully
	go func() {
		defer wg.Done()
		<-sigChan // Wait for a signal
		fmt.Println("Received signal, stopping Cromwell process...")
		err := cmd.Process.Signal(os.Interrupt) // Send interrupt signal to the process
		if err != nil {
			fmt.Println("Failed to send interrupt signal to Cromwell process:", err)
		}
		err = cmd.Wait() // Wait for the process to exit
		if err != nil {
			fmt.Println("Cromwell process exited with error:", err)
		}
		fmt.Println("Cromwell process stopped.")
	}()

	// Wait for the goroutine to exit
	wg.Wait()

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
	defer resp.Body.Close()
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
	defer resp.Body.Close()

	file, err := os.Create(cromwellFileName)
	if err != nil {
		return err
	}
	defer file.Close()

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
		err := os.MkdirAll(p, 0755)
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

func createDefaultConfig(db MysqlConfig, config string, port, maxConcurrentJobs int) error {
	viper.SetConfigType("json")

	// Set the configuration values
	viper.Set("backend.default", "Local")
	viper.Set("backend.providers.Local.actor-factory", "cromwell.backend.impl.sfs.config.ConfigBackendLifecycleActorFactory")
	viper.Set("backend.providers.Local.config.max-concurrent-workflows", 1)
	viper.Set("backend.providers.Local.config.concurrent-job-limit", maxConcurrentJobs)
	viper.Set("backend.providers.Local.config.filesystems.local.localization", []string{"hard-link", "soft-link", "copy"})

	viper.Set("webservice.port", port)

	viper.Set("database.profile", "slick.jdbc.MySQLProfile$")
	viper.Set("database.db.driver", "com.mysql.cj.jdbc.Driver")
	viper.Set("database.db.url", fmt.Sprintf("jdbc:mysql://%s:%d/cromwell?rewriteBatchedStatements=true&useSSL=false", db.Host, db.Port))
	viper.Set("database.db.user", db.Username)
	viper.Set("database.db.password", db.Password)
	viper.Set("database.db.connectionTimeout", 50000)

	viper.Set("call-caching.enabled", true)
	viper.Set("call-caching.invalidate-bad-cache-results", true)

	viper.Set("docker.perform-registry-lookup-if-digest-is-provided", false)

	// Write the configuration to file
	err := viper.WriteConfigAs(config)
	if err != nil {
		fmt.Printf("Error writing config file: %s", err)
		return err
	}

	fmt.Println("Config file written successfully")
	return nil
}

func checkMysqlConn(dbConf MysqlConfig) error {
	dbConn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", dbConf.Username, dbConf.Password, dbConf.Host, dbConf.Port)
	db, err := sql.Open("mysql", dbConn)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return err
	}
	return nil
}
