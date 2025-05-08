package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/thinkgos/gin-contrib/cache"
	redisStore "github.com/thinkgos/gin-contrib/cache/persist/redis"
)

func main() {
	app := gin.New()

	store := redisStore.NewStore(redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    "localhost:6379",
	}))

	app.GET("/hello",
		cache.Cache(
			store,
			5*time.Second,
			cache.WithGenerateKey(cache.GenerateRequestPath),
		),
		func(c *gin.Context) {
			c.String(200, "hello world")
		},
	)
	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
