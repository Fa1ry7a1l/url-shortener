package logger

import "go.uber.org/zap"

// Logger is the logging subset required by HTTP middleware.
type Logger interface {
	// Info writes an informational log entry.
	Info(msg string, fields ...zap.Field)
}

// New builds the production zap logger used by the service.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	return cfg.Build()
}
