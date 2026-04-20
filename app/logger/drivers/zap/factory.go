package zap_logger

import (
	"fuse_video_streamer/logger/interfaces"
)

type Factory struct {
	cache *loggerCache
}

var _ interfaces.LoggerFactory = &Factory{}

func NewFactory() *Factory {
	return &Factory{
		cache: newLoggerCache(),
	}
}

func (f *Factory) NewLogger(service string) (interfaces.Logger, error) {
	return newLogger(f.cache, service)
}
