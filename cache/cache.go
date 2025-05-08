package cache

import (
	"bytes"
	"context"
	"crypto/sha1"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"

	"github.com/thinkgos/gin-contrib/cache/persist"
)

// PageCachePrefix default page cache key prefix
var PageCachePrefix = "gincache.page.cache:"

// Logger logger interface
type Logger interface {
	Errorf(ctx context.Context, format string, args ...any)
}

// Encoding interface
type Encoding interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// Config for cache
type Config struct {
	// store the cache backend to store response
	store persist.Store
	// expire the cache expiration time
	expire time.Duration
	// rand duration for expire
	rand func() time.Duration
	// generate key for store, bool means need cache or not
	generateKey func(c *gin.Context) (string, bool)
	// group single flight group
	group *singleflight.Group
	// logger debug
	logger Logger
	// encoding default: JSONEncoding
	encode Encoding
}

// Option custom option
type Option func(c *Config)

// WithGenerateKey custom generate key ,default is GenerateRequestURIKey.
func WithGenerateKey(f func(c *gin.Context) (string, bool)) Option {
	return func(c *Config) {
		if f != nil {
			c.generateKey = f
		}
	}
}

// WithSingleflight custom single flight group, default is private single flight group.
func WithSingleflight(group *singleflight.Group) Option {
	return func(c *Config) {
		if group != nil {
			c.group = group
		}
	}
}

// WithRandDuration custom rand duration for expire, default return zero
// expiration time always expire + rand()
func WithRandDuration(rand func() time.Duration) Option {
	return func(c *Config) {
		if rand != nil {
			c.rand = rand
		}
	}
}

// WithLogger custom logger, default is Discard.
func WithLogger(l Logger) Option {
	return func(c *Config) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithEncoding custom Encoding, default is JSONEncoding.
func WithEncoding(encode Encoding) Option {
	return func(c *Config) {
		if encode != nil {
			c.encode = encode
		}
	}
}

// Cache user must pass store and store expiration time to cache and with custom option.
// default caching response with uri, which use PageCachePrefix
func Cache(store persist.Store, expire time.Duration, opts ...Option) gin.HandlerFunc {
	cfg := Config{
		store:       store,
		expire:      expire,
		rand:        func() time.Duration { return 0 },
		generateKey: GenerateRequestUri,
		group:       new(singleflight.Group),
		logger:      NewDiscard(),
		encode:      JSONEncoding{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(c *gin.Context) {
		key, needCache := cfg.generateKey(c)
		if !needCache {
			c.Next()
			return
		}

		// read cache first
		bodyCache := poolGet()
		defer poolPut(bodyCache)
		bodyCache.encoding = cfg.encode

		if err := cfg.store.Get(key, bodyCache); err != nil {
			// BodyWriter in order to dup the response
			bodyWriter := &BodyWriter{ResponseWriter: c.Writer}
			c.Writer = bodyWriter

			inFlight := false
			// use single flight to avoid Hotspot Invalid
			bc, _, shared := cfg.group.Do(key, func() (any, error) {
				c.Next()
				inFlight = true
				bc := getBodyCacheFromBodyWriter(bodyWriter, cfg.encode)
				if !c.IsAborted() && bodyWriter.Status() < 300 && bodyWriter.Status() >= 200 {
					if err = cfg.store.Set(key, bc, cfg.expire+cfg.rand()); err != nil {
						cfg.logger.Errorf(c.Request.Context(), "set cache key error: %s, cache key: %s", err, key)
					}
				}
				return bc, nil
			})
			if !inFlight && shared {
				c.Abort()
				responseWithBodyCache(c, bc.(*BodyCache))
			}
		} else {
			c.Abort()
			responseWithBodyCache(c, bodyCache)
		}
	}
}

// GenerateKeyWithPrefix generate key with GenerateKeyWithPrefix and u,
// if key is larger than 200,it will use sha1.Sum
// key like: prefix+u or prefix+sha1(u)
func GenerateKeyWithPrefix(prefix, key string) string {
	if len(key) > 200 {
		d := sha1.Sum([]byte(key))
		return prefix + string(d[:])
	}
	return prefix + key
}

// GenerateRequestUri generate key with PageCachePrefix and request uri
func GenerateRequestUri(c *gin.Context) (string, bool) {
	return GenerateKeyWithPrefix(PageCachePrefix, url.QueryEscape(c.Request.RequestURI)), true
}

// GenerateRequestPath generate key with PageCachePrefix and request Path
func GenerateRequestPath(c *gin.Context) (string, bool) {
	return GenerateKeyWithPrefix(PageCachePrefix, url.QueryEscape(c.Request.URL.Path)), true
}

// BodyWriter dup response writer body
type BodyWriter struct {
	gin.ResponseWriter
	dupBody bytes.Buffer
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *BodyWriter) Write(b []byte) (int, error) {
	w.dupBody.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString the string into the response body.
func (w *BodyWriter) WriteString(s string) (int, error) {
	w.dupBody.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func getBodyCacheFromBodyWriter(writer *BodyWriter, encode Encoding) *BodyCache {
	return &BodyCache{
		writer.Status(),
		writer.Header().Clone(),
		writer.dupBody.Bytes(),
		encode,
	}
}

func responseWithBodyCache(c *gin.Context, bodyCache *BodyCache) {
	c.Writer.WriteHeader(bodyCache.Status)
	for k, v := range bodyCache.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Writer.Write(bodyCache.Data) // nolint: errcheck
}
