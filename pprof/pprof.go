package pprof

import (
	"expvar"
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

const (
	// defaultPrefix url prefix of pprof
	defaultPrefix = "/debug/pprof"
)

// Router the standard HandlerFuncs from the net/http/pprof package with
// the provided gin.Engine. prefixOptions is a optional. If not prefixOptions,
// the default path prefix("/debug/pprof") is used, otherwise first prefixOptions will be path prefix.
// "/debug/vars" use for expvar
func Router(g gin.IRouter, prefixOptions ...string) {
	prefix := defaultPrefix
	if len(prefixOptions) > 0 {
		prefix = prefixOptions[0]
	}
	r := g.Group(prefix)
	{
		r.GET("/", gin.WrapF(pprof.Index))
		r.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		r.GET("/profile", gin.WrapF(pprof.Profile))
		r.POST("/symbol", gin.WrapF(pprof.Symbol))
		r.GET("/symbol", gin.WrapF(pprof.Symbol))
		r.GET("/trace", gin.WrapF(pprof.Trace))
		r.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		r.GET("/block", gin.WrapH(pprof.Handler("block")))
		r.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		r.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		r.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		r.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
	}
	g.GET("/debug/vars", gin.WrapH(expvar.Handler()))
}
