package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
)

const CromwellUrl = "https://github.com/broadinstitute/cromwell/releases/download/85/cromwell-85.jar"

func StartCromwellServer() error {
	docker := isInUserPath("docker")
	if !docker {
		return fmt.Errorf("docker is not installed. please install docker first")
	}

	java := isInUserPath("java")
	if !java {
		return fmt.Errorf("java is not installed. please install java first")
	}

	// check if it has internet connection
	_, err := http.Get("https://www.google.com")
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
	logsPath := filepath.Dir(cromwell)
	err = startCromwellProcess(cromwell, logsPath)
	if err != nil {
		return err
	}
	return nil
}

func startCromwellProcess(cromwellPath, logsPath string) error {
	cmd := exec.Command("java", "-DLOG_MODE=pretty", "-jar", cromwellPath, "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return err
	}

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		return err
	}
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
