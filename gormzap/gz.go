package gormzap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thinkgos/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Logger logger for gorm2
type Logger struct {
	log *logger.Log
	gormlogger.Config
}

// Option logger/recover option
type Option func(l *Logger)

// WithConfig optional custom logger.Config
func WithConfig(cfg gormlogger.Config) Option {
	return func(l *Logger) {
		l.Config = cfg
	}
}

// SetGormDBLogger set db logger
func SetGormDBLogger(db *gorm.DB, l gormlogger.Interface) {
	db.Logger = l
}

// New logger form gorm2
func New(log *logger.Log, opts ...Option) gormlogger.Interface {
	log.AddCallerSkipPackage(
		"gorm.io/gorm",
		"github.com/thinkgos/gin-contrib/gormzap",
	)
	l := &Logger{
		log: log,
		Config: gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			LogLevel:                  gormlogger.Warn,
		},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LogMode log mode
func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info print info
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= gormlogger.Info && l.log.Enabled(logger.InfoLevel) {
		l.log.OnDebugContext(ctx).Printf(msg, args...)
	}
}

// Warn print warn messages
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= gormlogger.Warn && l.log.Enabled(logger.WarnLevel) {
		l.log.OnWarnContext(ctx).Printf(msg, args...)
	}
}

// Error print error messages
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= gormlogger.Error && l.log.Enabled(logger.ErrorLevel) {
		l.log.OnErrorContext(ctx).Printf(msg, args...)
	}
}

// Trace print sql message
func (l *Logger) Trace(ctx context.Context, begin time.Time, f func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil &&
		l.LogLevel >= gormlogger.Error &&
		l.log.Enabled(logger.ErrorLevel) &&
		(!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := f()
		l.log.OnErrorContext(ctx).
			Error(err).
			Duration("latency", elapsed).
			DoHookFunc(func(ctx context.Context) logger.Field {
				if rows == -1 {
					return logger.String("rows", "-")
				} else {
					return logger.Int64("rows", rows)
				}
			}).
			String("sql", sql).
			Msg("trace")
	case elapsed > l.SlowThreshold &&
		l.SlowThreshold != 0 &&
		l.LogLevel >= gormlogger.Warn &&
		l.log.Enabled(logger.WarnLevel):
		sql, rows := f()
		l.log.OnErrorContext(ctx).
			Error(err).
			String("slow!!!", fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)).
			Duration("latency", elapsed).
			DoHookFunc(func(ctx context.Context) logger.Field {
				if rows == -1 {
					return logger.String("rows", "-")
				} else {
					return logger.Int64("rows", rows)
				}
			}).
			String("sql", sql).
			Msg("trace")
	case l.LogLevel == gormlogger.Info && l.log.Enabled(logger.InfoLevel):
		sql, rows := f()
		l.log.OnInfoContext(ctx).
			Error(err).
			Duration("latency", elapsed).
			DoHookFunc(func(ctx context.Context) logger.Field {
				if rows == -1 {
					return logger.String("rows", "-")
				} else {
					return logger.Int64("rows", rows)
				}
			}).
			String("sql", sql).
			Msg("trace")
	}
}
