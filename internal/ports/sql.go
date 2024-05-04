package ports

type Sql interface {
	CheckConnection() error
}
