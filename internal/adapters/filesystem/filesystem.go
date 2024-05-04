package filesystem

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"github.com/lmtani/pumbaa/internal/ports"
)

type LocalFilesystem struct {
	l ports.Logger
}

func NewLocalFilesystem(l ports.Logger) *LocalFilesystem {
	return &LocalFilesystem{l: l}
}

func (l *LocalFilesystem) CreateDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0750)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *LocalFilesystem) HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func (l *LocalFilesystem) MoveFile(srcPath, destPath string) error {
	return os.Rename(srcPath, destPath)
}

func (l *LocalFilesystem) ReadFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		l.l.Error(err.Error())
		return "", err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		l.l.Error(err.Error())
		return "", err
	}
	return string(contents), nil
}

func (l *LocalFilesystem) WriteFile(path, contents string) error {
	file, err := os.Create(path)
	if err != nil {
		l.l.Error(err.Error())
		return err
	}
	defer file.Close()

	_, err = file.WriteString(contents)
	if err != nil {
		l.l.Error(err.Error())
		return err
	}
	return nil
}

// CreateZip creates a zip file at the specified destination path containing all the files listed in filePaths.
func (l *LocalFilesystem) CreateZip(destinationPath string, filePaths []string) error {
	// Create a file to write the zip archive to.
	outFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Create a new zip archive writer.
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// Loop through each file that needs to be added to the zip.
	for _, filePath := range filePaths {
		// Open the file to be archived.
		fileToZip, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		// Get the file information.
		info, err := fileToZip.Stat()
		if err != nil {
			return err
		}

		// Create a zip file header based on the file information.
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.Base(filePath)
		header.Method = zip.Deflate // Set the compression method.

		// Create a writer for the file in the zip archive.
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Copy the file data to the zip writer.
		if _, err = io.Copy(writer, fileToZip); err != nil {
			return err
		}
	}

	// Make sure to check for errors from closing the zip writer.
	return zipWriter.Close()
}

func (l *LocalFilesystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
