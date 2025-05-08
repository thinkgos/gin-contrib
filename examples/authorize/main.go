package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/thinkgos/gin-contrib/authorize"
)

type Account struct {
	Type     string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
}

func main() {
	auth, err := authorize.New[*Account](authorize.Config{
		Timeout:        time.Hour * 24,
		RefreshTimeout: time.Hour * (24 + 1),
		Lookup:         "header:Authorization:Bearer",
		Algorithm:      "HS256",
		Key:            []byte("testSecretKey"),
		PrivKey:        "",
		PubKey:         "",
		Issuer:         "gin-contrib",
	})
	if err != nil {
		panic(err)
	}
	router := gin.New()
	router.Use(
		auth.Middleware(
			authorize.WithSkip(func(c *gin.Context) bool {
				return c.Request.URL.Path == "/login"
			}),
			authorize.WithUnauthorizedFallback(func(c *gin.Context, err error) {
				c.String(http.StatusUnauthorized, err.Error())
			}),
		))
	// curl -v http://127.0.0.1:8080/login -X 'POST' \
	//   --header 'Accept: */*' \
	//   --header 'Content-Type: application/json' \
	//   --header 'Accept-Encoding: gzip, deflate, br' \
	//   --header 'Connection: keep-alive' \
	//   --data-raw '{}' --insecure
	router.POST("/login", func(c *gin.Context) {
		tk, expiresAt, err := auth.GenerateToken(&authorize.Claims[*Account]{
			RegisteredClaims: jwt.RegisteredClaims{
				ID:      "1123",
				Subject: "1123",
			},
			Meta: &Account{
				Type:     "test",
				Username: "test",
			},
		})
		if err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, map[string]any{
			"code":       http.StatusOK,
			"token":      tk,
			"expires_at": expiresAt.Unix(),
			"message":    "ok",
		})
	})
	// curl -v http://127.0.0.1:8080/test-auth -X 'GET' \
	//   --header 'Accept: */*' \
	//   --header 'Accept-Encoding: gzip, deflate, br' \
	//   --header 'Connection: keep-alive' \
	//   --header 'Authorization: Bearer {{realtoken}}' \
	//   --insecure
	router.GET("/test-auth", func(c *gin.Context) {
		c.String(http.StatusOK, "success auth")
	})

	router.Run(":8080") // nolint: errcheck
}
