package netx

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestEchoServer(t *testing.T) {
	for _, test := range []struct {
		network string
		address string
	}{
		{"unix", "/tmp/echo-server.sock"},
		{"tcp", "127.0.0.1:56789"},
	} {
		test := test
		t.Run(test.address, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			server := &Server{
				Addr:    test.address,
				Context: ctx,
				Handler: &Echo{},
			}

			done := &sync.WaitGroup{}
			done.Add(1)
			go func() {
				defer done.Done()
				server.ListenAndServe()
			}()

			// Give a bit of time to the server to bind the socket.
			time.Sleep(50 * time.Millisecond)
			join := &sync.WaitGroup{}

			for i := 0; i != 10; i++ {
				join.Add(1)
				go func() {
					defer join.Done()

					conn, err := net.Dial(test.network, test.address)
					if err != nil {
						t.Error(err)
						return
					}
					defer conn.Close()

					b := [12]byte{}

					if _, err := conn.Write([]byte("Hello World!")); err != nil {
						t.Error(err)
						return
					}

					if _, err := io.ReadFull(conn, b[:]); err != nil {
						t.Error(err)
						return
					}

					if s := string(b[:]); s != "Hello World!" {
						t.Error(s)
						return
					}
				}()
			}

			join.Wait()
			cancel()
			done.Wait()
		})
	}
}
