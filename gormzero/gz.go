package gormzero

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Logger logger for gorm2
type Logger struct {
	log *zerolog.Logger
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
func New(log *zerolog.Logger, opts ...Option) gormlogger.Interface {
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
	l.log.Debug().Enabled()
	if l.LogLevel >= gormlogger.Info {
		l.log.Debug().Ctx(ctx).Msgf(msg, args...)
	}
}

// Warn print warn messages
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= gormlogger.Warn {
		l.log.Warn().Ctx(ctx).Msgf(msg, args...)
	}
}

// Error print error messages
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= gormlogger.Error {
		l.log.Error().Ctx(ctx).Msgf(msg, args...)
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
		(!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := f()
		l.log.Error().Ctx(ctx).
			Err(err).
			Dur("latency", elapsed).
			Func(func(e *zerolog.Event) {
				if rows == -1 {
					e.Str("rows", "-")
				} else {
					e.Int64("rows", rows)
				}

			}).
			Str("sql", sql).
			Msg("trace")
	case elapsed > l.SlowThreshold &&
		l.SlowThreshold != 0 &&
		l.LogLevel >= gormlogger.Warn:
		sql, rows := f()
		l.log.Error().Ctx(ctx).
			Err(err).
			Str("slow!!!", fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)).
			Dur("latency", elapsed).
			Func(func(e *zerolog.Event) {
				if rows == -1 {
					e.Str("rows", "-")
				} else {
					e.Int64("rows", rows)
				}
			}).
			Str("sql", sql).
			Msg("trace")
	case l.LogLevel == gormlogger.Info:
		sql, rows := f()
		l.log.Info().Ctx(ctx).
			Err(err).
			Dur("latency", elapsed).
			Func(func(e *zerolog.Event) {
				if rows == -1 {
					e.Str("rows", "-")
				} else {
					e.Int64("rows", rows)
				}
			}).
			Str("sql", sql).
			Msg("trace")
	}
}
