// Package mock provides shared no-op logger implementations for tests.
package mock

import (
	interfaces_logger "fuse_video_streamer/logger/interfaces"
)

// NoopLogger satisfies interfaces_logger.Logger by discarding all output.
type NoopLogger struct{}

func (NoopLogger) Info(string)         {}
func (NoopLogger) Warn(string)         {}
func (NoopLogger) Error(string, error) {}
func (NoopLogger) Fatal(string, error) {}
func (NoopLogger) Debug(string)        {}

var _ interfaces_logger.Logger = NoopLogger{}

// NoopLoggerFactory satisfies interfaces_logger.LoggerFactory, returning
// NoopLogger instances.
type NoopLoggerFactory struct{}

func (NoopLoggerFactory) NewLogger(_ string) (interfaces_logger.Logger, error) {
	return NoopLogger{}, nil
}

var _ interfaces_logger.LoggerFactory = NoopLoggerFactory{}
