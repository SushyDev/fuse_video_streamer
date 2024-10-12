package logger

import (
	"go.uber.org/zap"
)

var Logger *zap.SugaredLogger

func init() {
	logger, _ := zap.NewDevelopment()

	Logger = logger.Sugar()
}
