package ports

type Wdl interface {
	GetDependencies(contents string) ([]string, error)
	ReplaceImports(contents string) (string, error)
}
