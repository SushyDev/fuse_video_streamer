package zap_logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fuse_video_streamer/logger/interfaces"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var LogDir = "logs"

var loggers = make(map[string]*zap.SugaredLogger)

func createLogger(fileName string) (*zap.SugaredLogger, error) {
	filePath := filepath.Join(LogDir, fileName)

	// Create the log directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, err
	}

	// Open the log file
	logFile, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
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

func getLogger(fileName string) (*zap.SugaredLogger, error) {
	if logger, ok := loggers[fileName]; ok {
		return logger, nil
	}

	logger, err := createLogger(fileName)
	if err != nil {
		return nil, err
	}

	loggers[fileName] = logger

	return logger, nil
}

type Logger struct {
	logger  *zap.SugaredLogger
	service string
}

var _ interfaces.Logger = &Logger{}

func NewLogger(service string) (*Logger, error) {
	filename := strings.ToLower(strings.ReplaceAll(service, " ", "_"))

	logger, err := getLogger(filename + ".log")
	if err != nil {
		return nil, err
	}

	return &Logger{
		logger:  logger,
		service: service,
	}, nil
}

func (instance *Logger) Info(message string) {
	loggerMessage := strings.ReplaceAll(message, "\t", " ")
	instance.logger.Info(loggerMessage)

	formattedMessage := fmt.Sprintf("INFO	%s:	%s", instance.service, message)
	log.Println(formattedMessage)
}

func (instance Logger) Warn(message string) {
	loggerMessage := strings.ReplaceAll(message, "\t", " ")
	instance.logger.Warn(loggerMessage)

	formattedMessage := fmt.Sprintf("WARN	%s:	%s", instance.service, message)
	log.Println(formattedMessage)
}

func (instance Logger) Error(message string, err error) {
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
	loggerMessage := strings.ReplaceAll(message, "\t", " ")
	instance.logger.Debug(loggerMessage)

	formattedMessage := fmt.Sprintf("DEBUG	%s:	%s", instance.service, message)
	log.Println(formattedMessage)
}

func (instance *Logger) Close() {
	instance.logger.Sync()

	delete(loggers, instance.service)
}
