package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/thinkgos/gin-contrib/traceid"
)

func main() {
	router := gin.New()
	router.Use(traceid.TraceId())
	router.GET("/", func(c *gin.Context) {
		fmt.Println(traceid.FromTraceId(c.Request.Context()))
		fmt.Println(traceid.GetTraceId(c))
	})
	router.Run(":8080")// nolint: errcheck
}
