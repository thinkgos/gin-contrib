package main

import (
	"context"
	"time"

	"github.com/thinkgos/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/thinkgos/gin-contrib/gormzap"
)

func main() {
	l := logger.NewLogger()
	log := gormzap.New(
		l.WithNewHook(
			&logger.ImmutableString{Key: "service", Value: "test"},
			logger.HookFunc(func(ctx context.Context) logger.Field {
				v := ctx.Value("requestId")
				if v == nil {
					return logger.Skip()
				}
				if vv, ok := v.(string); ok {
					return logger.String("requestId", vv)
				}
				return logger.Skip()
			})).
			SetNewCallerCore(logger.NewCallerCore()),
		gormzap.WithConfig(gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  false,
			IgnoreRecordNotFoundError: false,
			LogLevel:                  gormlogger.Info,
		}),
	)
	// your dialector
	db, _ := gorm.Open(nil, &gorm.Config{Logger: log})
	// do your things
	_ = db
}
