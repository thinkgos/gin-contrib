package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/thinkgos/gin-contrib/cache/persist"
)

// Store redis store
type Store struct {
	Redisc *redis.Client
}

// NewStore new redis store
func NewStore(client *redis.Client) *Store {
	return &Store{client}
}

// Set implement persist.Store interface
func (store *Store) Set(key string, value any, expire time.Duration) error {
	return store.Redisc.Set(context.Background(), key, value, expire).Err()
}

// Get implement persist.Store interface
func (store *Store) Get(key string, value any) error {
	err := store.Redisc.Get(context.Background(), key).Scan(value)
	if err != nil {
		if err == redis.Nil {
			return persist.ErrCacheMiss
		}
		return err
	}
	return nil
}

// Delete implement persist.Store interface
func (store *Store) Delete(key string) error {
	return store.Redisc.Del(context.Background(), key).Err()
}
