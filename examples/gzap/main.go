package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thinkgos/logger"

	"github.com/thinkgos/gin-contrib/gzap"
)

func main() {
	r := gin.New()

	l := logger.NewLogger()

	// Add a ginzap middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	r.Use(gzap.Logger(
		l.WithNewHook(&logger.ImmutableString{Key: "app", Value: "example"}).
			SetNewCallerCore(logger.NewCallerCore()),
		gzap.WithCustomFields(
			func(c *gin.Context) logger.Field { return logger.String("custom field1", c.ClientIP()) },
			func(c *gin.Context) logger.Field { return logger.String("custom field2", c.ClientIP()) },
		),
		gzap.WithSkipLogging(func(c *gin.Context) bool { return c.Request.URL.Path == "/skiplogging" }),
		gzap.WithEnableBody(true),
	))

	// Logs all panic to error log
	//   - stack means whether output the stack info.
	r.Use(gzap.Recovery(
		l.WithNewHook(&logger.ImmutableString{Key: "app", Value: "example"}),
		true,
		gzap.WithCustomFields(
			func(c *gin.Context) logger.Field { return logger.String("custom field1", c.ClientIP()) },
			func(c *gin.Context) logger.Field { return logger.String("custom field2", c.ClientIP()) },
		),
	))

	// Example ping request.
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example when panic happen.
	r.GET("/panic", func(c *gin.Context) {
		panic("An unexpected error happen!")
	})

	r.GET("/error", func(c *gin.Context) {
		c.Error(errors.New("An error happen 1")) // nolint: errcheck,staticcheck
		c.Error(errors.New("An error happen 2")) // nolint: errcheck,staticcheck
	})

	r.GET("/skiplogging", func(c *gin.Context) {
		c.String(200, "i am skip logging, log should be not output")
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080") // nolint: errcheck
}
