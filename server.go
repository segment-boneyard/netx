package netx

import (
	"context"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

// ListenAndServe listens on the address addr and then call Serve to handle
// the incoming connections.
func ListenAndServe(addr string, handler Handler) error {
	return (&Server{
		Addr:    addr,
		Handler: handler,
	}).ListenAndServe()
}

// Serve accepts incoming connections on the Listener lstn, creating a new
// service goroutine for each. The service goroutines simply invoke the
// handler's ServeConn method.
func Serve(lstn net.Listener, handler Handler) error {
	return (&Server{
		Handler: handler,
	}).Serve(lstn)
}

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

// A Server defines parameters for running servers that accept connections over
// TCP or unix domains.
type Server struct {
	Addr     string          // address to listen on
	Handler  Handler         // handler to invoke on new connections
	ErrorLog *log.Logger     // the logger used to output internal errors
	Context  context.Context // the base context used by the server
}

// ListenAndServe listens on the server address and then call Serve to handle
// the incoming connections.
func (s *Server) ListenAndServe() (err error) {
	var lstn net.Listener

	if lstn, err = Listen(s.Addr); err == nil {
		err = s.Serve(lstn)
	}

	return
}

// Serve accepts incoming connections on the Listener lstn, creating a new
// service goroutine for each. The service goroutines simply invoke the
// handler's ServeConn method.
func (s *Server) Serve(lstn net.Listener) (err error) {
	defer lstn.Close()

	join := &sync.WaitGroup{}
	defer join.Wait()

	ctx := s.Context
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func(ctx context.Context) {
		<-ctx.Done()
		closeRead(lstn)
	}(ctx)

	const maxBackoff = 1 * time.Second
	for {
		var conn net.Conn

		for attempt := 0; true; attempt++ {
			if conn, err = lstn.Accept(); err == nil {
				break
			}
			if !IsTemporary(err) {
				break
			}

			// Backoff strategy for handling temporary errors, this prevents from
			// retrying too fast when errors like running out of file descriptors
			// occur.
			backoff := time.Duration(attempt*attempt) * 10 * time.Millisecond
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			s.logf("Accept error: %v; retrying in %v", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
		}

		if err != nil {
			select {
			default:
			case <-ctx.Done():
				// Don't report errors when the server stopped because its
				// context was canceled.
				err = nil
			}
			return
		}

		join.Add(1)
		go s.serve(ctx, conn, join)
	}
}

func (s *Server) serve(ctx context.Context, conn net.Conn, join *sync.WaitGroup) {
	defer func() { Recover(recover(), conn, s.ErrorLog) }()

	defer join.Done()
	defer conn.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.Handler.ServeConn(ctx, conn)
}

func (s *Server) logf(format string, args ...interface{}) {
	logf(s.ErrorLog)(format, args...)
}

// Recover is intended to be used by servers that gracefully handle panics from
// their handlers.
func Recover(err interface{}, conn net.Conn, logger *log.Logger) {
	if err == nil {
		return
	}

	logf := logf(logger)
	laddr := conn.LocalAddr()
	raddr := conn.RemoteAddr()

	if e, ok := err.(error); ok {
		logf("error serving %s->%s: %v", laddr, raddr, e)
	} else {
		buf := make([]byte, 262144)
		buf = buf[:runtime.Stack(buf, false)]
		logf("panic serving %s->%s: %v\n%s", laddr, raddr, err, string(buf))
	}
}

func logf(logger *log.Logger) func(string, ...interface{}) {
	if logger == nil {
		return log.Printf
	}
	return logger.Printf
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

	// Pass is the implementation of a connection that does nothing.
	Pass Handler = HandlerFunc(func(ctx context.Context, conn net.Conn) {
		// do nothing
	})
)
