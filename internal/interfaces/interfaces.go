package interfaces

import (
	"context"
	"github.com/lmtani/pumbaa/internal/entities"
	"golang.org/x/oauth2"
	"net/http"
)

type Filesystem interface {
	CreateDirectory(dir string) error
	MoveFile(srcPath, destPath string) error
	HomeDir() (string, error)
	ReadFile(path string) (string, error)
	WriteFile(path, contents string) error
	CreateZip(destinationPath string, filePaths []string) error
	FileExists(path string) bool
}

type GoogleCloudPlatform interface {
	GetStorageClient(ctx context.Context) (CloudStorageClient, error)
	GetIAPToken(ctx context.Context, aud string) (string, error)
}

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
	Head(url string) (resp *http.Response, err error)
	DownloadWithProgress(url string, fileName string) error
}

type Logger interface {
	Info(msg string)
	Warning(msg string)
	Error(msg string)
}

type Prompt interface {
	SelectByKey(taskOptions []string) (string, error)
	SelectByIndex(sfn func(input string, index int) bool, items interface{}) (int, error)
}

type CromwellServer interface {
	Kill(o string) (entities.SubmitResponse, error)
	Status(o string) (entities.SubmitResponse, error)
	Outputs(o string) (entities.OutputsResponse, error)
	Query(params *entities.ParamsQueryGet) (entities.QueryResponse, error)
	Metadata(o string, params *entities.ParamsMetadataGet) (entities.MetadataResponse, error)
	Submit(wdl, inputs, dependencies, options string) (entities.SubmitResponse, error)
}

type Sql interface {
	CheckConnection() error
}

type Wdl interface {
	GetDependencies(contents string) ([]string, error)
	ReplaceImports(contents string) (string, error)
}

type Writer interface {
	Primary(string)
	Accent(string)
	Message(string)
	Error(string)
	Table(table Table)
	QueryTable(table entities.QueryResponse)
	ResourceTable(table entities.TotalResources)
	MetadataTable(d entities.MetadataResponse) error
	Json(interface{}) error
}

// CloudStorageClient defines the interface for storage client operations needed
type CloudStorageClient interface {
	Close() error
}

// DependencyFactory defines the interface for creating dependencies
type DependencyFactory interface {
	NewStorageClient(ctx context.Context) (CloudStorageClient, error)
	NewTokenSource(ctx context.Context, aud string) (oauth2.TokenSource, error)
}

type Table interface {
	Header() []string
	Rows() [][]string
}
