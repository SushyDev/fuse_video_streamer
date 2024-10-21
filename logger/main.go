package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

var AppLogPath = "logs/app.log"
var StreamLogPath = "logs/stream.log"
var FuseLogPath = "logs/fuse.log"
var ApiLogPath = "logs/api.log"

var loggers = make(map[string]*zap.SugaredLogger)

func createLogger(filePath string) (*zap.SugaredLogger, error) {
	// Create the log directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
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

func GetLogger(filePath string) (*zap.SugaredLogger, error) {
    if logger, ok := loggers[filePath]; ok {
        return logger, nil
    }

    logger, err := createLogger(filePath)
    if err != nil {
        return nil, err
    }

    loggers[filePath] = logger

    return logger, nil
}
