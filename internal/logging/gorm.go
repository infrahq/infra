package logging

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/logger"
)

type gormLogger struct {
	l *zap.SugaredLogger
}

func ToGormLogger(l *zap.SugaredLogger) logger.Interface {
	return &gormLogger{l: l}
}

func (l *gormLogger) LogMode(ll logger.LogLevel) logger.Interface {
	var lvl zapcore.LevelEnabler
	switch ll {
	case logger.Silent:
		lvl = zapcore.DPanicLevel
	case logger.Error:
		lvl = zapcore.ErrorLevel
	case logger.Warn:
		lvl = zapcore.WarnLevel
	case logger.Info:
		lvl = zapcore.InfoLevel
	}

	return ToGormLogger(newLogger(lvl, os.Stderr).Sugar())
}

func (l *gormLogger) Info(ctx context.Context, s string, args ...interface{}) {
	l.l.Infof(s, args...)
}

func (l *gormLogger) Warn(ctx context.Context, s string, args ...interface{}) {
	l.l.Warnf(s, args...)
}

func (l *gormLogger) Error(ctx context.Context, s string, args ...interface{}) {
	l.l.Errorf(s, args...)
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// don't log traces for now
	_, _ = fc()
}
