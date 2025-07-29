package zap_logger

import (
	"fuse_video_streamer/logger/interfaces"
)

type Factory struct{}

var _ interfaces.LoggerFactory = &Factory{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) NewLogger(service string) (interfaces.Logger, error) {
	logger, err := NewLogger(service)
	if err != nil {
		return nil, err
	}

	return logger, nil
}
