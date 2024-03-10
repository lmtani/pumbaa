package ports

type GoogleCloudStorage interface {
	GetClient() (interface{}, error)
}
