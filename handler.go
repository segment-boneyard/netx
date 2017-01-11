package netx

import (
	"bufio"
	"context"
	"net"
	"time"
)

// A Handler manages a network connection.
//
// The ServeConn method is called by a Server when a new client connection is
// established, the method receives the connection and a context object that
// the server may use to indicate that it's shutting down.
//
// Servers recover from panics that escape the handlers and log the error and
// stack trace.
type Handler interface {
	ServeConn(ctx context.Context, conn net.Conn)
}

// The HandlerFunc type allows simple functions to be used as connection
// handlers.
type HandlerFunc func(context.Context, net.Conn)

// ServeConn calls f.
func (f HandlerFunc) ServeConn(ctx context.Context, conn net.Conn) {
	f(ctx, conn)
}

var (
	// Echo is the implementation of a connection handler that simply sends what
	// it receives back to the client.
	Echo Handler = HandlerFunc(func(ctx context.Context, conn net.Conn) {
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
		Copy(conn, conn)
	})

	// EchoLine is the implementation of a connection handler that reads '\n'
	// terminated lines and echos them back to the client, expecting the client
	// not to send more than one line before getting it echoed back.
	//
	// The implementation supports cancellations and ensures that no partial
	// lines are read from the connection.
	//
	// The maximum line length is limited to 8192 bytes.
	EchoLine Handler = HandlerFunc(func(ctx context.Context, conn net.Conn) {
		r := bufio.NewReaderSize(conn, 8192)

		for {
			select {
			default:
			case <-ctx.Done():
				return
			}

			conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			if _, err := r.Peek(1); err != nil {
				if IsTimeout(err) {
					continue
				}
			}

			line, prefix, err := r.ReadLine()

			if prefix || err != nil {
				conn.Close()
				return
			}

			if line = line[:len(line)+1]; line[len(line)-1] == '\r' {
				line = line[:len(line)+1]
			}

			if _, err := conn.Write(line); err != nil {
				return
			}
		}
	})

	// Pass is the implementation of a connection that does nothing.
	Pass Handler = HandlerFunc(func(ctx context.Context, conn net.Conn) {
		// do nothing
	})
)
