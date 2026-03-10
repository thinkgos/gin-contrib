package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/thinkgos/gin-contrib/gzero"
)

type ctxClientIp struct{}

func main() {
	r := gin.New()

	lg := log.Logger.With().
		Str("app", "example").
		Logger().
		Hook(zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
			clientIp := e.GetCtx().Value(ctxClientIp{}).(string)
			e.Str("custom field1", clientIp).
				Str("custom field2", clientIp)
		}))

	r.Use(gzero.Logger(
		new(lg),
		gzero.WithSkipLogging(func(c *gin.Context) bool { return c.Request.URL.Path == "/skiplogging" }),
		gzero.WithEnableBody(true),
	))
	r.Use(gzero.Recovery(
		new(lg),
		true,
	))
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxClientIp{}, c.ClientIP()))
	})

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
