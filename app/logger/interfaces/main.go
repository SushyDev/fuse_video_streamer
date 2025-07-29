package interfaces

type Logger interface {
	Info(message string)
	Warn(message string)
	Error(message string, err error)
	Fatal(message string, err error)
	Debug(message string)
}

type LoggerFactory interface {
	NewLogger(service string) (Logger, error)
}
