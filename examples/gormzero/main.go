package main

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/thinkgos/gin-contrib/gormzero"
)

func main() {
	l := log.Logger

	lg := l.With().Str("service", "test").Logger().Hook(zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
		v := e.GetCtx().Value("requestId")
		if v == nil {
			return
		}
		if vv, ok := v.(string); ok {
			e.Str("requestId", vv)
		}
	}))
	log := gormzero.New(
		new(lg),
		gormzero.WithConfig(gormlogger.Config{
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
