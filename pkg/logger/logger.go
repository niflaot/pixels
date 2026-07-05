package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a zap logger from logger configuration.
func New(config Config) (*zap.Logger, error) {
	var level zapcore.Level

	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	if config.Format != FormatConsole && config.Format != FormatJSON {
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.Encoding = string(config.Format)
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if config.Format == FormatConsole {
		zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return zapConfig.Build()
}
