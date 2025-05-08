package redis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/thinkgos/gin-contrib/cache/persist"
)

// Test typical cache interactions
func Test_Memory_typicalGetSet(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	defer mr.Close()

	storeCache := NewStore(redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	}))

	value := "foo"
	err = storeCache.Set("value", value, time.Hour)
	require.NoError(t, err)

	value = ""
	err = storeCache.Get("value", &value)
	require.NoError(t, err)
	require.Equal(t, "foo", value)
}

func Test_Memory_Expiration(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	storeCache := NewStore(redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	}))

	value := 10

	// Test Set w/ short time
	err = storeCache.Set("int", value, time.Second)
	require.NoError(t, err)
	mr.FastForward(time.Second)
	err = storeCache.Get("int", &value)
	require.ErrorIs(t, err, persist.ErrCacheMiss)

	// Test Set w/ longer time.
	err = storeCache.Set("int", value, time.Hour)
	require.NoError(t, err)
	mr.FastForward(time.Second)
	err = storeCache.Get("int", &value)
	require.NoError(t, err)

	// Test Set w/ forever.
	err = storeCache.Set("int", value, -1)
	require.NoError(t, err)
	mr.FastForward(time.Second)
	err = storeCache.Get("int", &value)
	require.NoError(t, err)
}

func Test_Memory_Empty(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	storeCache := NewStore(redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	}))

	err = storeCache.Get("notexist", time.Hour)
	require.Error(t, err)
	require.ErrorIs(t, err, persist.ErrCacheMiss)

	err = storeCache.Delete("notexist")
	require.NoError(t, err)
}
