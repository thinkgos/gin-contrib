package main

import (
	"time"

	"github.com/gin-gonic/gin"
	inmemory "github.com/patrickmn/go-cache"

	"github.com/thinkgos/gin-contrib/cache"
	"github.com/thinkgos/gin-contrib/cache/persist/memory"
)

func main() {
	app := gin.New()

	app.GET("/hello/:a/:b",
		cache.Cache(
			memory.NewStore(inmemory.New(time.Minute, time.Minute*10)),
			5*time.Second,
			cache.WithGenerateKey(func(c *gin.Context) (string, bool) {
				a := c.Param("a")
				b := c.Param("b")
				return cache.GenerateKeyWithPrefix(cache.PageCachePrefix, a+":"+b), true
			}),
		),
		func(c *gin.Context) {
			c.String(200, "hello world")
		})
	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
