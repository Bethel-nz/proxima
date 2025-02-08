package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout", "proxima.log"}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
	}

	return config.Build(zapcore.NewConsoleEncoder(encoderConfig))
}
