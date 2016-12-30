package netx

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/nettest"
)

func TestServerConn(t *testing.T) {
	nettest.TestConn(t, func() (c1 net.Conn, c2 net.Conn, stop func(), err error) {
		cnch := make(chan net.Conn)
		done := make(chan struct{})

		n, a, f := listenAndServe(HandlerFunc(func(ctx context.Context, conn net.Conn) {
			cnch <- conn
			<-done
		}))

		if c1, err = net.Dial(n, a); err != nil {
			return
		}

		c2 = <-cnch

		stop = func() {
			close(done)
			c2.Close()
			c1.Close()
			f()
		}
		return
	})
}

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

func listenAndServe(h Handler) (net string, addr string, close func()) {
	lstn, err := Listen("127.0.0.1:0")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go (&Server{
		Handler:  h,
		Context:  ctx,
		ErrorLog: log.New(os.Stderr, "listen: ", 0),
	}).Serve(lstn)

	net, addr, close = lstn.Addr().Network(), lstn.Addr().String(), cancel
	return
}
