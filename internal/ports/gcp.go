package ports

type GoogleCloudPlatform interface {
	GetStorageClient() (interface{}, error)
	GetIAPToken() (string, error)
}
