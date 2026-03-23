package logger

import "go.uber.org/zap"

type Logger interface {
	Info(msg string, fields ...zap.Field)
}

func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	return cfg.Build()
}
