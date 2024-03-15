package ports

type Filesystem interface {
	CreateDirectory(dir string) error
	MoveFile(srcPath, destPath string) error
	ZipFiles(workflowPath, zipPath string, files []string) ([]string, error)
	ReplaceImports(workflowPath string) (string, error)
	IsInUserPath(path string) bool
	HomeDir() (string, error)
}
