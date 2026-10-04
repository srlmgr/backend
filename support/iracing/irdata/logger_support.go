package irdata

import (
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	"github.com/srlmgr/backend/log"
)

type customLeveledLogger struct {
	logger *log.Logger
}

var _ retryablehttp.LeveledLogger = (*customLeveledLogger)(nil)

func newCustomLeveledLogger(logger *log.Logger) *customLeveledLogger {
	return &customLeveledLogger{logger: logger}
}

func (l *customLeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, l.kvToFields(keysAndValues...)...)
}

func (l *customLeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, l.kvToFields(keysAndValues...)...)
}

func (l *customLeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, l.kvToFields(keysAndValues...)...)
}

func (l *customLeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, l.kvToFields(keysAndValues...)...)
}

//nolint:funlen // many cases
func (l *customLeveledLogger) kvToFields(kv ...interface{}) []zap.Field {
	if len(kv) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(kv)/2)

	for i := 0; i < len(kv); i += 2 {
		// Handle odd number of arguments
		if i+1 >= len(kv) {
			fields = append(
				fields,
				zap.Any("invalid_key", kv[i]),
			)
			break
		}

		keyRaw := kv[i]
		value := kv[i+1]

		// If already a zap.Field, append directly
		if field, ok := keyRaw.(zap.Field); ok {
			fields = append(fields, field)
			i-- // adjust because this consumes only one slot
			continue
		}

		key, ok := keyRaw.(string)
		if !ok {
			key = fmt.Sprintf("%v", keyRaw)
		}

		// Special-case errors for better encoding
		switch v := value.(type) {
		case error:
			fields = append(fields, zap.Error(v))
		case zap.Field:
			fields = append(fields, v)
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case int64:
			fields = append(fields, zap.Int64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))

		default:
			fields = append(fields, zap.Any(key, v))
		}
	}

	return fields
}
