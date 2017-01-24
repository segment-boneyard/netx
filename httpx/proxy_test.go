package httpx

import (
	"net/http"
	"testing"

	"github.com/segmentio/netx"
	"github.com/segmentio/netx/httpx/httpxtest"
)

func TestProxy(t *testing.T) {
	httpxtest.TestServer(t, func(config httpxtest.ServerConfig) (string, func()) {
		origin, closeOrigin := listenAndServe(&Server{
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			MaxHeaderBytes: config.MaxHeaderBytes,
			Handler:        config.Handler,
		})

		proxy, closeProxy := listenAndServe(&Server{
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			MaxHeaderBytes: config.MaxHeaderBytes,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				_, req.URL.Host = netx.SplitNetAddr(origin)
				(&ReverseProxy{}).ServeHTTP(w, req)
			}),
		})

		return proxy, func() {
			closeProxy()
			closeOrigin()
		}
	})
}
