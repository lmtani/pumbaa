package ports

type Logger interface {
	Info(msg string)
	Warning(msg string)
	Error(msg string)
}
