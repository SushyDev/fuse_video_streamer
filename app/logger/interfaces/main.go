// Package interfaces defines the logging abstraction used throughout the application.
package interfaces

// Logger provides structured logging with different severity levels.
type Logger interface {
	Info(message string)
	Warn(message string)
	Error(message string, err error)
	Fatal(message string, err error)
	Debug(message string)
}

// LoggerFactory creates Logger instances with a service name for context.
type LoggerFactory interface {
	NewLogger(service string) (Logger, error)
}
