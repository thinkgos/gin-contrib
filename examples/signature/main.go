package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thinkgos/gin-contrib/signature"
	httpsign "github.com/thinkgos/http-signature-go"
)

func main() {
	// Init define secret params
	readKeyID := httpsign.KeyId("read")
	writeKeyID := httpsign.KeyId("write")

	// Init server
	r := gin.Default()

	parser := httpsign.NewParser(
		httpsign.WithSigningMethods(
			httpsign.SigningMethodHmacSha256.Alg(),
			func() httpsign.SigningMethod { return httpsign.SigningMethodHmacSha256 },
		),
		httpsign.WithSigningMethods(
			httpsign.SigningMethodHmacSha512.Alg(),
			func() httpsign.SigningMethod { return httpsign.SigningMethodHmacSha512 },
		),
	)

	parser.AddMetadata(readKeyID, httpsign.Metadata{ // nolint: errcheck
		Scheme: 0,
		Alg:    httpsign.SigningMethodHmacSha256.Alg(),
		Key:    []byte("HMACSHA256-SecretKey"),
	})
	parser.AddMetadata(writeKeyID, httpsign.Metadata{ // // nolint: errcheck
		Scheme: 0,
		Alg:    httpsign.SigningMethodHmacSha512.Alg(),
		Key:    []byte("HMACSHA512-SecretKey"),
	})
	//Create middleware with default rule. Could modify by parse Option func
	auth := signature.Authenticator{
		Parser: parser,
	}

	r.Use(auth.Authenticated())
	r.GET("/a", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/b", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/c", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	r.Run(":8080") // nolint: errcheck
}
