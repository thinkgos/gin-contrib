package cache

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"

	"github.com/thinkgos/gin-contrib/cache/persist"
	"github.com/thinkgos/gin-contrib/cache/persist/memory"
	redisStore "github.com/thinkgos/gin-contrib/cache/persist/redis"
)

var longLengthThan200Key = "/" + strings.Repeat("qwertyuiopasdfghjklzxcvbnm", 8)
var enableRedis = false

var newStore = func(defaultExpiration time.Duration) persist.Store {
	if enableRedis {
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379"
		}
		return redisStore.NewStore(redis.NewClient(&redis.Options{
			Addr: redisHost + ":" + port,
		}))
	}
	return memory.NewStore(cache.New(defaultExpiration, time.Minute*10))
}

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(target string, router *gin.Engine) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestCache(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache/ping", Cache(store, time.Second*3), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest("/cache/ping", r)
	w2 := performRequest("/cache/ping", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheNoNeedCache(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache/ping",
		Cache(store,
			time.Second*3,
			WithGenerateKey(func(c *gin.Context) (string, bool) {
				return "", false
			}),
			WithRandDuration(func() time.Duration {
				return time.Duration(rand.Intn(5)) * time.Second
			}),
			WithSingleflight(&singleflight.Group{}),
			WithLogger(NewDiscard()),
			WithEncoding(JSONEncoding{}),
		),
		func(c *gin.Context) {
			c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().UnixNano()))
		},
	)

	w1 := performRequest("/cache/ping", r)
	time.Sleep(time.Millisecond * 10)
	w2 := performRequest("/cache/ping", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheExpire(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache/ping", Cache(store, time.Second), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest("/cache/ping", r)
	time.Sleep(time.Second * 3)
	w2 := performRequest("/cache/ping", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFile(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.LoadHTMLFiles("../testdata/template.html")
	r.GET("/cache/html", Cache(store, time.Second*3), func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"value": fmt.Sprint(time.Now().UnixNano())})
	})

	w1 := performRequest("/cache/html", r)
	w2 := performRequest("/cache/html", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheHtmlFileExpire(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.LoadHTMLFiles("../testdata/template.html")
	r.GET("/cache/html", Cache(store, time.Second*1), func(c *gin.Context) {
		c.HTML(http.StatusOK, "template.html", gin.H{"value": fmt.Sprint(time.Now().UnixNano())})
	})

	w1 := performRequest("/cache/html", r)
	time.Sleep(time.Second * 3)
	w2 := performRequest("/cache/html", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheAborted(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache/aborted", Cache(store, time.Second*3), func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusOK, map[string]int64{"time": time.Now().UnixNano()})
	})

	w1 := performRequest("/cache/aborted", r)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("/cache/aborted", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheStatus400(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache/400", Cache(store, time.Second*3), func(c *gin.Context) {
		c.String(http.StatusBadRequest, fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest("/cache/400", r)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("/cache/400", r)

	assert.Equal(t, http.StatusBadRequest, w1.Code)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.NotEqual(t, w1.Body.String(), w2.Body.String())
}

func TestCacheStatus207(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache/207", Cache(store, time.Second*3), func(c *gin.Context) {
		c.String(http.StatusMultiStatus, fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest("/cache/207", r)
	time.Sleep(time.Millisecond * 500)
	w2 := performRequest("/cache/207", r)

	assert.Equal(t, http.StatusMultiStatus, w1.Code)
	assert.Equal(t, http.StatusMultiStatus, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheLongKey(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET(longLengthThan200Key, Cache(store, time.Second*3), func(c *gin.Context) {
		c.String(http.StatusOK, fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest(longLengthThan200Key, r)
	w2 := performRequest(longLengthThan200Key, r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheWithRequestPath(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache_with_path", Cache(store, time.Second*3, WithGenerateKey(GenerateRequestPath)), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest("/cache_with_path?foo=1", r)
	w2 := performRequest("/cache_with_path?foo=2", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
}

func TestCacheWithRequestURI(t *testing.T) {
	store := newStore(time.Second * 60)

	r := gin.New()
	r.GET("/cache_with_uri", Cache(store, time.Second*5), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().UnixNano()))
	})

	w1 := performRequest("/cache_with_uri?foo=1", r)
	w2 := performRequest("/cache_with_uri?foo=1", r)
	time.Sleep(time.Millisecond * 10)
	w3 := performRequest("/cache_with_uri?foo=2", r)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, w1.Body.String(), w2.Body.String())
	assert.NotEqual(t, w2.Body.String(), w3.Body.String())
}

type memoryDelayStore struct {
	*memory.Store
}

func newDelayStore(c *cache.Cache) *memoryDelayStore {
	return &memoryDelayStore{memory.NewStore(c)}
}

func (c *memoryDelayStore) Set(key string, value any, expires time.Duration) error {
	time.Sleep(time.Millisecond * 3)
	return c.Store.Set(key, value, expires)
}

func TestCacheInSingleflight(t *testing.T) {
	store := newDelayStore(cache.New(60*time.Second, time.Minute*10))

	r := gin.New()
	r.GET("/singleflight", Cache(store, time.Second*5), func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	outp := make(chan string, 10)

	for i := 0; i < 5; i++ {
		go func() {
			resp := performRequest("/singleflight", r)
			outp <- resp.Body.String()
		}()
	}
	time.Sleep(time.Millisecond * 500)
	for i := 0; i < 5; i++ {
		go func() {
			resp := performRequest("/singleflight", r)
			outp <- resp.Body.String()
		}()
	}
	time.Sleep(time.Millisecond * 500)

	for i := 0; i < 10; i++ {
		v := <-outp
		assert.Equal(t, "OK", v)
	}
}

func TestBodyWrite(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writer := &BodyWriter{c.Writer, bytes.Buffer{}}
	c.Writer = writer

	c.Writer.WriteHeader(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
	c.Writer.WriteString("foo") // nolint: errcheck
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Equal(t, "foo", w.Body.String())
	assert.Equal(t, "foo", writer.dupBody.String())
	assert.True(t, c.Writer.Written())
	c.Writer.WriteString("bar") // nolint: errcheck
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Equal(t, "foobar", w.Body.String())
	assert.Equal(t, "foobar", writer.dupBody.String())
	assert.True(t, c.Writer.Written())
}

func TestDiscard(_ *testing.T) {
	l := NewDiscard()
	l.Errorf(context.Background(), "")
}

func TestJSONEncoding(t *testing.T) {
	want := BodyCache{
		Status:   2,
		Header:   nil,
		Data:     []byte{1, 20, 3, 90},
		encoding: nil,
	}

	encode := JSONEncoding{}

	data, err := encode.Marshal(want)
	require.NoError(t, err)

	got := BodyCache{}
	err = encode.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestJSONGzipEncoding(t *testing.T) {
	want := BodyCache{
		Status:   2,
		Header:   nil,
		Data:     []byte{1, 20, 3, 90},
		encoding: nil,
	}

	encode := JSONGzipEncoding{}

	data, err := encode.Marshal(want)
	require.NoError(t, err)

	got := BodyCache{}
	err = encode.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
