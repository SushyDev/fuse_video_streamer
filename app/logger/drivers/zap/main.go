package zap_logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fuse_video_streamer/logger/interfaces"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var LogDir = "logs"

func createLogger(fileName string) (*zap.SugaredLogger, error) {
	filePath := filepath.Join(LogDir, fileName)

	// Create the log directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Open the log file
	logFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Define the core for the logger
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), // Use JSON encoding for structured logs
		zapcore.AddSync(logFile),                                 // Write logs to file
		zap.InfoLevel,                                            // Set the log level
	)

	// Return the new logger
	logger := zap.New(core, zap.AddCaller())

	return logger.Sugar(), nil
}

type Logger struct {
	logger  *zap.SugaredLogger
	service string

	debugLogsEnabled bool

	// cache is the Factory cache this logger was obtained from, used for cleanup.
	cache *loggerCache
}

var _ interfaces.Logger = &Logger{}

func (instance *Logger) Info(message string) {
	loggerMessage := strings.ReplaceAll(message, "\t", " ")
	instance.logger.Info(loggerMessage)

	formattedMessage := fmt.Sprintf("INFO	%s:	%s", instance.service, message)
	log.Println(formattedMessage)
}

func (instance *Logger) Warn(message string) {
	loggerMessage := strings.ReplaceAll(message, "\t", " ")
	instance.logger.Warn(loggerMessage)

	formattedMessage := fmt.Sprintf("WARN	%s:	%s", instance.service, message)
	log.Println(formattedMessage)
}

func (instance *Logger) Error(message string, err error) {
	loggerMessage := fmt.Sprintf("%s: %v", message, err)
	instance.logger.Error(loggerMessage)

	formattedMessage := fmt.Sprintf("ERROR	%s:	%s: %v", instance.service, message, err)
	log.Println(formattedMessage)
}

func (instance *Logger) Fatal(message string, err error) {
	loggerMessage := fmt.Sprintf("%s: %v", message, err)
	instance.logger.Fatal(loggerMessage)

	formattedMessage := fmt.Sprintf("FATAL	%s:	%s: %v", instance.service, message, err)
	log.Fatal(formattedMessage)
}

func (instance *Logger) Debug(message string) {
	if !instance.debugLogsEnabled {
		return
	}

	loggerMessage := strings.ReplaceAll(message, "\t", " ")
	instance.logger.Debug(loggerMessage)

	formattedMessage := fmt.Sprintf("DEBUG	%s:	%s", instance.service, message)
	log.Println(formattedMessage)
}

func (instance *Logger) Close() {
	instance.logger.Sync()
	instance.cache.remove(instance.service)
}

// loggerCache holds a per-Factory cache of named SugaredLoggers, eliminating
// the package-level global state that made testing difficult.
type loggerCache struct {
	mu      sync.RWMutex
	loggers map[string]*zap.SugaredLogger
}

func newLoggerCache() *loggerCache {
	return &loggerCache{
		loggers: make(map[string]*zap.SugaredLogger),
	}
}

func (c *loggerCache) get(fileName string) (*zap.SugaredLogger, error) {
	c.mu.RLock()
	logger, ok := c.loggers[fileName]
	c.mu.RUnlock()

	if ok {
		return logger, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	if logger, ok := c.loggers[fileName]; ok {
		return logger, nil
	}

	logger, err := createLogger(fileName)
	if err != nil {
		return nil, err
	}

	c.loggers[fileName] = logger

	return logger, nil
}

func (c *loggerCache) remove(service string) {
	c.mu.Lock()
	delete(c.loggers, service)
	c.mu.Unlock()
}

// newLogger creates a Logger backed by the given cache.
func newLogger(cache *loggerCache, service string) (*Logger, error) {
	filename := strings.ToLower(strings.ReplaceAll(service, " ", "_"))

	logger, err := cache.get(filename + ".log")
	if err != nil {
		return nil, err
	}

	debug := true

	return &Logger{
		logger:           logger,
		service:          service,
		debugLogsEnabled: debug,
		cache:            cache,
	}, nil
}
