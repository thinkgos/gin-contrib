package gormzap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/thinkgos/gin-contrib/internal/pool"
)

// Logger logger for gorm2
type Logger struct {
	log *zap.Logger
	logger.Config
	customFields []func(ctx context.Context) zap.Field
	callerCore   *CallerCore
}

// Option logger/recover option
type Option func(l *Logger)

// WithCustomFields optional custom field
func WithCustomFields(fields ...func(ctx context.Context) zap.Field) Option {
	return func(l *Logger) {
		l.customFields = fields
	}
}

// WithConfig optional custom logger.Config
func WithConfig(cfg logger.Config) Option {
	return func(l *Logger) {
		l.Config = cfg
	}
}

// WithSkipPackages optional custom logger.Config
func WithSkipPackages(skipPackages ...string) Option {
	return func(l *Logger) {
		l.callerCore.AddSkipPackage(skipPackages...)
	}
}

// WithConfig optional custom logger.Config
func WithCallerCore(c *CallerCore) Option {
	return func(l *Logger) {
		if c != nil {
			l.callerCore = c
		}
	}
}

// SetGormDBLogger set db logger
func SetGormDBLogger(db *gorm.DB, l logger.Interface) {
	db.Logger = l
}

// New logger form gorm2
func New(zapLogger *zap.Logger, opts ...Option) logger.Interface {
	l := &Logger{
		log: zapLogger,
		Config: logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			LogLevel:                  logger.Warn,
		},
		callerCore: NewCallerCore(),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LogMode log mode
func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info print info
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= logger.Info && l.log.Level().Enabled(zap.InfoLevel) {
		msg = fmt.Sprintf(msg, args...)
		if neeCaller := l.callerCore.caller != nil && l.callerCore.Enabled(zap.InfoLevel); neeCaller || len(l.customFields) > 0 {
			fc := pool.Get()
			defer pool.Put(fc)
			for _, customField := range l.customFields {
				fc.Fields = append(fc.Fields, customField(ctx))
			}
			if neeCaller {
				fc.Fields = append(fc.Fields, l.callerCore.caller(l.callerCore.skip, l.callerCore.skipPackages...))
			}
			l.log.Debug(msg, fc.Fields...)
		} else {
			l.log.Debug(msg)
		}
	}
}

// Warn print warn messages
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= logger.Warn && l.log.Level().Enabled(zap.WarnLevel) {
		msg = fmt.Sprintf(msg, args...)
		if neeCaller := l.callerCore.caller != nil && l.callerCore.Enabled(zap.WarnLevel); neeCaller || len(l.customFields) > 0 {
			fc := pool.Get()
			defer pool.Put(fc)
			for _, customField := range l.customFields {
				fc.Fields = append(fc.Fields, customField(ctx))
			}
			if neeCaller {
				fc.Fields = append(fc.Fields, l.callerCore.caller(l.callerCore.skip, l.callerCore.skipPackages...))
			}
			l.log.Warn(msg, fc.Fields...)
		} else {
			l.log.Warn(msg)
		}
	}
}

// Error print error messages
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	if l.LogLevel >= logger.Error && l.log.Level().Enabled(zap.ErrorLevel) {
		msg = fmt.Sprintf(msg, args...)
		if neeCaller := l.callerCore.caller != nil && l.callerCore.Enabled(zap.ErrorLevel); neeCaller || len(l.customFields) > 0 {
			fc := pool.Get()
			defer pool.Put(fc)
			for _, customField := range l.customFields {
				fc.Fields = append(fc.Fields, customField(ctx))
			}
			if neeCaller {
				fc.Fields = append(fc.Fields, l.callerCore.caller(l.callerCore.skip, l.callerCore.skipPackages...))
			}
			l.log.Error(msg, fc.Fields...)
		} else {
			l.log.Error(msg)
		}
	}
}

// Trace print sql message
func (l *Logger) Trace(ctx context.Context, begin time.Time, f func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil &&
		l.LogLevel >= logger.Error &&
		l.log.Level().Enabled(zap.ErrorLevel) &&
		(!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		fc := pool.Get()
		defer pool.Put(fc)
		for _, customField := range l.customFields {
			fc.Fields = append(fc.Fields, customField(ctx))
		}
		fc.Fields = append(fc.Fields,
			zap.Error(err),
			zap.Duration("latency", elapsed),
		)
		if l.callerCore.caller != nil && l.callerCore.Enabled(zap.ErrorLevel) {
			fc.Fields = append(fc.Fields, l.callerCore.caller(l.callerCore.skip, l.callerCore.skipPackages...))
		}

		sql, rows := f()
		if rows == -1 {
			fc.Fields = append(fc.Fields, zap.String("rows", "-"))
		} else {
			fc.Fields = append(fc.Fields, zap.Int64("rows", rows))
		}
		fc.Fields = append(fc.Fields, zap.String("sql", sql))
		l.log.Error("trace", fc.Fields...)
	case elapsed > l.SlowThreshold &&
		l.SlowThreshold != 0 &&
		l.LogLevel >= logger.Warn &&
		l.log.Level().Enabled(zap.WarnLevel):
		fc := pool.Get()
		defer pool.Put(fc)
		for _, customField := range l.customFields {
			fc.Fields = append(fc.Fields, customField(ctx))
		}
		fc.Fields = append(fc.Fields, zap.Error(err))
		if l.callerCore.caller != nil && l.callerCore.Enabled(zap.WarnLevel) {
			fc.Fields = append(fc.Fields, l.callerCore.caller(l.callerCore.skip, l.callerCore.skipPackages...))
		}
		fc.Fields = append(fc.Fields,
			zap.String("slow!!!", fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)),
			zap.Duration("latency", elapsed),
		)

		sql, rows := f()
		if rows == -1 {
			fc.Fields = append(fc.Fields, zap.String("rows", "-"))
		} else {
			fc.Fields = append(fc.Fields, zap.Int64("rows", rows))
		}
		fc.Fields = append(fc.Fields, zap.String("sql", sql))
		l.log.Warn("trace", fc.Fields...)
	case l.LogLevel == logger.Info && l.log.Level().Enabled(zap.InfoLevel):
		fc := pool.Get()
		defer pool.Put(fc)
		for _, customField := range l.customFields {
			fc.Fields = append(fc.Fields, customField(ctx))
		}
		fc.Fields = append(fc.Fields, zap.Error(err))
		if l.callerCore.caller != nil && l.callerCore.Enabled(zap.InfoLevel) {
			fc.Fields = append(fc.Fields, l.callerCore.caller(l.callerCore.skip, l.callerCore.skipPackages...))
		}
		fc.Fields = append(fc.Fields, zap.Duration("latency", elapsed))
		sql, rows := f()
		if rows == -1 {
			fc.Fields = append(fc.Fields, zap.String("rows", "-"))
		} else {
			fc.Fields = append(fc.Fields, zap.Int64("rows", rows))
		}
		fc.Fields = append(fc.Fields, zap.String("sql", sql))
		l.log.Info("trace", fc.Fields...)
	}
}

// Immutable custom immutable field
// Deprecated: use Any instead
func Immutable(key string, value any) func(ctx context.Context) zap.Field {
	return Any(key, value)
}

// Any custom immutable any field
func Any(key string, value any) func(ctx context.Context) zap.Field {
	field := zap.Any(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// String custom immutable string field
func String(key string, value string) func(ctx context.Context) zap.Field {
	field := zap.String(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// Int64 custom immutable int64 field
func Int64(key string, value int64) func(ctx context.Context) zap.Field {
	field := zap.Int64(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// Uint64 custom immutable uint64 field
func Uint64(key string, value uint64) func(ctx context.Context) zap.Field {
	field := zap.Uint64(key, value)
	return func(ctx context.Context) zap.Field { return field }
}

// Float64 custom immutable float32 field
func Float64(key string, value float64) func(ctx context.Context) zap.Field {
	field := zap.Float64(key, value)
	return func(ctx context.Context) zap.Field { return field }
}
