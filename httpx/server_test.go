package httpx

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/segmentio/netx"
	"github.com/segmentio/netx/httpx/httpxtest"
)

func TestServer(t *testing.T) {
	httpxtest.TestServer(t, func(config httpxtest.ServerConfig) (string, func()) {
		return listenAndServe(&Server{
			Handler:        config.Handler,
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			MaxHeaderBytes: config.MaxHeaderBytes,
		})
	})
}

func listenAndServe(h netx.Handler) (url string, close func()) {
	lstn, err := netx.Listen("127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go (&netx.Server{
		Handler:  h,
		Context:  ctx,
		ErrorLog: log.New(os.Stderr, "listen: ", 0),
	}).Serve(lstn)

	url, close = "http://"+lstn.Addr().String(), cancel
	return
}
