package caddy_logger_loki

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// logger is a zap logger wrapper that implements the go-kit log.logger interface
type logger struct {
	logger *zap.Logger
}

// newLogger creates a new logger
func newLogger(zapLogger *zap.Logger) logger {
	return logger{logger: zapLogger}
}

// Log implements the go-kit log.logger interface
func (l logger) Log(keyvals ...interface{}) error {
	if len(keyvals)%2 != 0 {
		return fmt.Errorf("invalid number of keyvals")
	}

	level := "info"
	fields := make([]zapcore.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			return fmt.Errorf("keyvals must be a sequence of key/value pairs")
		}
		value := keyvals[i+1]
		fields = append(fields, zap.Any(key, value))
	}

	switch level {
	case "debug":
		l.logger.Debug("", fields...)
	case "info":
		l.logger.Info("", fields...)
	case "warn":
		l.logger.Warn("", fields...)
	case "error":
		l.logger.Error("", fields...)
	default:
		l.logger.Info("", fields...)
	}

	return nil
}
