// Package gzero provides log handling using zerolog package.
package gzero

import (
	"bytes"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/thinkgos/httpcurl"
)

// Option logger/recover option
type Option func(c *Config)

// WithSkipLogging optional custom skip logging option.
func WithSkipLogging(f func(c *gin.Context) bool) Option {
	return func(c *Config) {
		if f != nil {
			c.skipLogging = f
		}
	}
}

// WithEnableBody optional custom enable request/response body.
func WithEnableBody(b bool) Option {
	return func(c *Config) {
		c.enableBody.Store(b)
	}
}

// WithExternalEnableBody optional custom enable request/response body control by external itself.
func WithExternalEnableBody(b *atomic.Bool) Option {
	return func(c *Config) {
		if b != nil {
			c.enableBody = b
		}
	}
}

// WithBodyLimit optional custom request/response body limit.
// default: <=0, mean not limit
func WithBodyLimit(limit int) Option {
	return func(c *Config) {
		c.limit = limit
	}
}

// WithSkipRequestBody optional custom skip request body logging option.
func WithSkipRequestBody(f func(c *gin.Context) bool) Option {
	return func(c *Config) {
		if f != nil {
			c.skipRequestBody = f
		}
	}
}

// WithSkipResponseBody optional custom skip response body logging option.
func WithSkipResponseBody(f func(c *gin.Context) bool) Option {
	return func(c *Config) {
		if f != nil {
			c.skipResponseBody = f
		}
	}
}

// WithUseLoggerLevel optional use logging level.
func WithUseLoggerLevel(f func(c *gin.Context) zerolog.Level) Option {
	return func(c *Config) {
		if f != nil {
			c.useLoggerLevel = f
		}
	}
}

func WithEnableDebugCurl(b bool) Option {
	return func(c *Config) {
		if b {
			c.debugCurl = httpcurl.New()
		} else {
			c.debugCurl = nil
		}
	}
}

// Config logger/recover config
type Config struct {
	// if returns true, it will skip logging.
	skipLogging func(c *gin.Context) bool
	// if returns true, it will skip request body.
	skipRequestBody func(c *gin.Context) bool
	// if returns true, it will skip response body.
	skipResponseBody func(c *gin.Context) bool
	// use zerolog log level,
	// default:
	// 	zerolog.ErrorLevel: when status >= http.StatusInternalServerError && status <= http.StatusNetworkAuthenticationRequired
	// 	zerolog.WarnLevel: when status >= http.StatusBadRequest && status <= http.StatusUnavailableForLegalReasons
	//  zerolog.InfoLevel: otherwise.
	useLoggerLevel func(c *gin.Context) zerolog.Level
	enableBody     *atomic.Bool       // enable request/response body
	limit          int                // <=0: mean not limit
	debugCurl      *httpcurl.HttpCurl // debug curl
}

func skipRequestBody(c *gin.Context) bool {
	v := c.Request.Header.Get("Content-Type")
	d, params, err := mime.ParseMediaType(v)
	if err != nil || (d != "multipart/form-data" && d != "multipart/mixed") {
		return false
	}
	_, ok := params["boundary"]
	return ok
}

func skipResponseBody(c *gin.Context) bool {
	// TODO: add skip response body rule
	return false
}

func useLoggerLevel(c *gin.Context) zerolog.Level {
	status := c.Writer.Status()
	if status >= http.StatusInternalServerError &&
		status <= http.StatusNetworkAuthenticationRequired {
		return zerolog.ErrorLevel
	}
	if status >= http.StatusBadRequest &&
		status <= http.StatusUnavailableForLegalReasons &&
		status != http.StatusUnauthorized {
		return zerolog.WarnLevel
	}
	return zerolog.InfoLevel
}

func newConfig() Config {
	return Config{
		skipLogging:      func(c *gin.Context) bool { return false },
		skipRequestBody:  func(c *gin.Context) bool { return false },
		skipResponseBody: func(c *gin.Context) bool { return false },
		useLoggerLevel:   useLoggerLevel,
		enableBody:       &atomic.Bool{},
		limit:            0,
	}
}

// Logger returns a gin.HandlerFunc (middleware) that logs requests using rs/zerolog.
//
// Requests with errors are logged using zerolog.Error().
// Requests without errors are logged using zerolog.Info().
func Logger(log *zerolog.Logger, opts ...Option) gin.HandlerFunc {
	cfg := newConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return func(c *gin.Context) {
		respBodyBuilder := &strings.Builder{}
		reqBody := "skip request body"
		debugCurl := ""
		hasSkipRequestBody := skipRequestBody(c) || cfg.skipRequestBody(c)

		if cfg.enableBody.Load() {
			c.Writer = &bodyWriter{ResponseWriter: c.Writer, dupBody: respBodyBuilder}
			if !hasSkipRequestBody {
				reqBodyBuf, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.String(http.StatusInternalServerError, err.Error())
					c.Abort()
					return
				}
				c.Request.Body.Close() // nolint: errcheck
				c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBuf))
				if cfg.limit > 0 && len(reqBodyBuf) >= cfg.limit {
					reqBody = "larger request body"
				} else {
					reqBody = string(reqBodyBuf)
				}
			}
		}
		if !hasSkipRequestBody && cfg.debugCurl != nil {
			debugCurl, _ = cfg.debugCurl.IntoCurl(c.Request)
		}

		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		defer func() {
			if cfg.skipLogging(c) {
				return
			}
			var level zerolog.Level

			if len(c.Errors) > 0 {
				level = zerolog.ErrorLevel
			} else {
				level = cfg.useLoggerLevel(c)
			}
			log.WithLevel(level).
				Ctx(c.Request.Context()).
				Int("status", c.Writer.Status()).
				Str("method", c.Request.Method).
				Str("path", path).
				Str("route", c.FullPath()).
				Str("query", query).
				Str("ip", c.ClientIP()).
				Str("user-agent", c.Request.UserAgent()).
				Dur("latency", time.Since(start)).
				Func(func(e *zerolog.Event) {
					if cfg.enableBody.Load() {
						respBody := "skip response body"
						if hasSkipResponseBody := skipResponseBody(c) || cfg.skipResponseBody(c); !hasSkipResponseBody {
							if cfg.limit > 0 && respBodyBuilder.Len() >= cfg.limit {
								respBody = "larger response body"
							} else {
								respBody = respBodyBuilder.String()
							}
						}
						e.Str("requestBody", reqBody).
							Str("responseBody", respBody)
					}
					if debugCurl != "" {
						e.Str("curl", debugCurl)
					}
					if len(c.Errors) > 0 {
						for _, err := range c.Errors {
							e.Err(err)
						}
					}
				}).
				Msg("logging")
		}()

		c.Next()
	}
}

// Recovery returns a gin.HandlerFunc (middleware)
// that recovers from any panics and logs requests using rs/zerolog.
// All errors are logged using zerolog.Error().
// stack means whether output the stack info.
// The stack info is easy to find where the error occurs but the stack info is too large.
func Recovery(log *zerolog.Logger, stack bool, opts ...Option) gin.HandlerFunc {
	cfg := newConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					log.Error().
						Ctx(c.Request.Context()).
						Any("error", err).
						RawJSON("request", httpRequest).
						Msg(c.Request.URL.Path)
					// If the connection is dead, we can't write a status to it.
					_ = c.Error(err.(error))
					c.Abort()
					return
				}
				log.Error().
					Ctx(c.Request.Context()).
					Any("error", err).
					RawJSON("request", httpRequest).
					Func(func(e *zerolog.Event) {
						if stack {
							e.Bytes("stack", debug.Stack())
						}
					}).
					Msg("recovery from panic")
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

type bodyWriter struct {
	gin.ResponseWriter
	dupBody *strings.Builder
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.dupBody.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyWriter) WriteString(s string) (int, error) {
	w.dupBody.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
