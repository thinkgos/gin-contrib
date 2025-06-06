package oteltraceid

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Config defines the config for TraceId middleware
type Config struct {
	traceIdHeader string
	spanIdHeader  string
}

// Option TraceId option
type Option func(*Config)

// WithTraceIdHeader optional request id header (default "X-Trace-Id")
func WithTraceIdHeader(s string) Option {
	return func(c *Config) {
		c.traceIdHeader = s
	}
}

// WithSpanIdHeader optional request id header (default "X-Span-Id")
func WithSpanIdHeader(s string) Option {
	return func(c *Config) {
		c.spanIdHeader = s
	}
}

// TraceId is a middleware that injects a trace id into the context of each
// request. if it is empty, set to write head
//   - traceIdHeader is the name of the HTTP Header which contains the trace id.
//     Exported so that it can be changed by developers. (default "X-Trace-Id")
//   - spanIdHeader is the name of the HTTP Header which contains the span id.
//     Exported so that it can be changed by developers. (default "X-Span-Id")
func TraceId(opts ...Option) gin.HandlerFunc {
	cc := &Config{
		traceIdHeader: "X-Trace-Id",
		spanIdHeader:  "X-Span-Id",
	}
	for _, opt := range opts {
		opt(cc)
	}
	return func(c *gin.Context) {
		sc := trace.SpanContextFromContext(c.Request.Context())
		// set response header
		c.Header(cc.traceIdHeader, sc.TraceID().String())
		c.Header(cc.spanIdHeader, sc.SpanID().String())
		c.Next()
	}
}

func FromTraceId(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}

func FromSpanId(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		return sc.SpanID().String()
	}
	return ""
}
