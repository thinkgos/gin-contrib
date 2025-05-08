// Package gzap provides log handling using zap package.
// Code structure based on ginrus package.
// see github.com/gin-contrib/zap
package gzap

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
	"github.com/thinkgos/httpcurl"
	"github.com/thinkgos/logger"
)

// Option logger/recover option
type Option func(c *Config)

// WithCustomFields optional custom field
func WithCustomFields(fields ...func(c *gin.Context) logger.Field) Option {
	return func(c *Config) {
		c.customFields = fields
	}
}

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
func WithUseLoggerLevel(f func(c *gin.Context) logger.Level) Option {
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
	customFields []func(c *gin.Context) logger.Field
	// if returns true, it will skip logging.
	skipLogging func(c *gin.Context) bool
	// if returns true, it will skip request body.
	skipRequestBody func(c *gin.Context) bool
	// if returns true, it will skip response body.
	skipResponseBody func(c *gin.Context) bool
	// use logger level,
	// default:
	// 	logger.ErrorLevel: when status >= http.StatusInternalServerError && status <= http.StatusNetworkAuthenticationRequired
	// 	logger.WarnLevel: when status >= http.StatusBadRequest && status <= http.StatusUnavailableForLegalReasons
	//  logger.InfoLevel: otherwise.
	useLoggerLevel func(c *gin.Context) logger.Level
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

func useLoggerLevel(c *gin.Context) logger.Level {
	status := c.Writer.Status()
	if status >= http.StatusInternalServerError &&
		status <= http.StatusNetworkAuthenticationRequired {
		return logger.ErrorLevel
	}
	if status >= http.StatusBadRequest &&
		status <= http.StatusUnavailableForLegalReasons &&
		status != http.StatusUnauthorized {
		return logger.WarnLevel
	}
	return logger.InfoLevel
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

// Logger returns a gin.HandlerFunc (middleware) that logs requests using uber-go/logger.
//
// Requests with errors are logged using logger.Error().
// Requests without errors are logged using logger.Info().
func Logger(log *logger.Log, opts ...Option) gin.HandlerFunc {
	log.AddCallerSkipPackage("github.com/thinkgos/gin-contrib")
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
			var level logger.Level

			if len(c.Errors) > 0 {
				level = logger.ErrorLevel
			} else {
				level = cfg.useLoggerLevel(c)
			}
			log.OnLevelContext(c.Request.Context(), level).
				Int("status", c.Writer.Status()).
				String("method", c.Request.Method).
				String("path", path).
				String("route", c.FullPath()).
				String("query", query).
				String("ip", c.ClientIP()).
				String("user-agent", c.Request.UserAgent()).
				Duration("latency", time.Since(start)).
				Configure(func(e *logger.Event) {
					if cfg.enableBody.Load() {
						respBody := "skip response body"
						if hasSkipResponseBody := skipResponseBody(c) || cfg.skipResponseBody(c); !hasSkipResponseBody {
							if cfg.limit > 0 && respBodyBuilder.Len() >= cfg.limit {
								respBody = "larger response body"
							} else {
								respBody = respBodyBuilder.String()
							}
						}
						e.String("requestBody", reqBody).
							String("responseBody", respBody)
					}
					for _, fieldFunc := range cfg.customFields {
						e.With(fieldFunc(c))
					}
					if debugCurl != "" {
						e.String("curl", debugCurl)
					}
					if len(c.Errors) > 0 {
						for _, err := range c.Errors {
							e.Error(err)
						}
					}
				}).
				Msg("logging")
		}()

		c.Next()
	}
}

// Recovery returns a gin.HandlerFunc (middleware)
// that recovers from any panics and logs requests using uber-go/logger.
// All errors are logged using logger.Error().
// stack means whether output the stack info.
// The stack info is easy to find where the error occurs but the stack info is too large.
func Recovery(log *logger.Log, stack bool, opts ...Option) gin.HandlerFunc {
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
					log.OnErrorContext(c.Request.Context()).
						Any("error", err).
						ByteString("request", httpRequest).
						Msg(c.Request.URL.Path)
					// If the connection is dead, we can't write a status to it.
					_ = c.Error(err.(error))
					c.Abort()
					return
				}
				log.OnErrorContext(c.Request.Context()).
					Any("error", err).
					ByteString("request", httpRequest).
					Configure(func(e *logger.Event) {
						for _, fieldFunc := range cfg.customFields {
							e.With(fieldFunc(c))
						}
						if stack {
							e.ByteString("stack", debug.Stack())
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

// Any custom immutable any field
func Any(key string, value any) func(c *gin.Context) logger.Field {
	field := logger.Any(key, value)
	return func(c *gin.Context) logger.Field { return field }
}

// String custom immutable string field
func String(key, value string) func(c *gin.Context) logger.Field {
	field := logger.String(key, value)
	return func(c *gin.Context) logger.Field { return field }
}

// Int64 custom immutable int64 field
func Int64(key string, value int64) func(c *gin.Context) logger.Field {
	field := logger.Int64(key, value)
	return func(c *gin.Context) logger.Field { return field }
}

// Uint64 custom immutable uint64 field
func Uint64(key string, value uint64) func(c *gin.Context) logger.Field {
	field := logger.Uint64(key, value)
	return func(c *gin.Context) logger.Field { return field }
}

// Float64 custom immutable float32 field
func Float64(key string, value float64) func(c *gin.Context) logger.Field {
	field := logger.Float64(key, value)
	return func(c *gin.Context) logger.Field { return field }
}
