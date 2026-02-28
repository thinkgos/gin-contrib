package memory

import (
	"reflect"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/thinkgos/gin-contrib/cache/persist"
)

// Store memory store
type Store struct {
	Cache *cache.Cache
}

// NewStore new memory store
func NewStore(c *cache.Cache) *Store {
	return &Store{c}
}

// Set implement persist.Store interface
func (c *Store) Set(key string, value any, expire time.Duration) error {
	c.Cache.Set(key, value, expire)
	return nil
}

// Get implement persist.Store interface
func (c *Store) Get(key string, value any) error {
	val, found := c.Cache.Get(key)
	if !found {
		return persist.ErrCacheMiss
	}

	v := reflect.ValueOf(value)
	if v.Type().Kind() == reflect.Pointer && v.Elem().CanSet() {
		v.Elem().Set(reflect.Indirect(reflect.ValueOf(val)))
	}
	return nil
}

// Delete implement persist.Store interface
func (c *Store) Delete(key string) error {
	c.Cache.Delete(key)
	return nil
}
