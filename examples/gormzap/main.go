package main

import (
	"time"

	"github.com/thinkgos/logger"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/thinkgos/gin-contrib/gormzap"
)

type CustomAppValue struct{}

func (c CustomAppValue) RunHook(e *logger.Event) {
	e.String("service", "test")
}

func main() {
	l := logger.NewLogger()
	log := gormzap.New(
		l.WithNewHook(
			CustomAppValue{},
			logger.HookFunc(func(e *logger.Event) {
				v := e.Context().Value("requestId")
				if v != nil {
					if vv, ok := v.(string); ok {
						e.String("requestId", vv)
					}
				}
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
