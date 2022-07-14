package logging

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	gormlogger "gorm.io/gorm/logger"
)

type DatabaseLogger struct {
	SlowThreshold time.Duration
}

func (l *DatabaseLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (*DatabaseLogger) Info(ctx context.Context, format string, v ...interface{}) {
	L.Info().Msgf(format, v...)
}

func (*DatabaseLogger) Warn(ctx context.Context, format string, v ...interface{}) {
	L.Warn().Msgf(format, v...)
}

func (*DatabaseLogger) Error(ctx context.Context, format string, v ...interface{}) {
	L.Error().Msgf(format, v...)
}

func (l *DatabaseLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	level := zerolog.TraceLevel

	elapsed := time.Since(begin)
	if err != nil && !errors.Is(err, gormlogger.ErrRecordNotFound) {
		level = zerolog.ErrorLevel
	} else if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		level = zerolog.WarnLevel
	}

	L.WithLevel(level).
		CallerSkipFrame(3).
		Int64("rows", rows).
		Str("query", sql).
		Dur("elapsed", time.Since(begin)).
		Err(err).
		Msg("")
}

func NewDatabaseLogger(slow time.Duration) *DatabaseLogger {
	return &DatabaseLogger{
		SlowThreshold: slow,
	}
}
