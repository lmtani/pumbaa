package ports

type Filesystem interface {
	CreateDirectory(dir string) error
	MoveFile(srcPath, destPath string) error
	HomeDir() (string, error)
	ReadFile(path string) (string, error)
	WriteFile(path, contents string) error
	CreateZip(destinationPath string, filePaths []string) error
}
