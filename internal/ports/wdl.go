package ports

type Wdl interface {
	GetDependencies(workflowPath string) ([][]string, error)
}
