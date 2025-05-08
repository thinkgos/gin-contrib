package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	inmemory "github.com/patrickmn/go-cache"

	"github.com/thinkgos/gin-contrib/cache"
	"github.com/thinkgos/gin-contrib/cache/persist/memory"
)

func main() {
	app := gin.New()

	app.GET("/hello",
		cache.Cache(
			memory.NewStore(inmemory.New(time.Minute, time.Minute*10)),
			5*time.Second,
		),
		func(c *gin.Context) {
			log.Println("dfadfadfadf")
			c.String(200, "hello world")
		},
	)
	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
