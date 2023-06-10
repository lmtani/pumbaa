package util

import (
	"os"
	"path/filepath"
)

func CromwellSavePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(home, ".cromwell")
	err = CreateDirectory(saveDir)
	if err != nil {
		return "", err
	}

	fileName := filepath.Join(saveDir, "cromwell.jar")
	return fileName, nil
}

func CreateDirectory(p string) error {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		err := os.MkdirAll(p, 0750)
		if err != nil {
			return err
		}
	}
	return nil
}
