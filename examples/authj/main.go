package main

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	"github.com/thinkgos/gin-contrib/authj"
)

func main() {
	// load the casbin model and policy from files, database is also supported.
	e, err := casbin.NewEnforcer("authj_model.conf", "authj_policy.csv")
	if err != nil {
		panic(err)
	}

	// define your router, and use the Casbin authj middleware.
	// the access that is denied by authj will return HTTP 403 error.
	router := gin.New()
	router.Use(
		func(c *gin.Context) {
			// set context subject
			authj.ContextWithSubject(c, "alice")
		},
		authj.Authorizer(e),
	)
	router.GET("/dataset1/resource1", func(c *gin.Context) {
		c.String(http.StatusOK, "alice own this resource")
	})
	router.GET("/dataset2/resource1", func(c *gin.Context) {
		c.String(http.StatusOK, "alice do not own this resource")
	})
	router.Run(":8080") // nolint: errcheck
}
