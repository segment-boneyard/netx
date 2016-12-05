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

	ctx, cancel := context.WithCancel(s.context())
	defer cancel()

	errf := func(err error, backoff time.Duration) {
		s.logf("Accept error: %v; retrying in %v", err, backoff)
	}

	for {
		var conn net.Conn

		if conn, err = Accept(lstn, errf); err != nil {
			return
		}

		join.Add(1)
		go s.serve(ctx, conn, join)
	}
}

func (s *Server) serve(ctx context.Context, conn net.Conn, join *sync.WaitGroup) {
	defer func(addr string) {
		if err := recover(); err != nil {
			s.recover(err, addr)
		}
	}(conn.RemoteAddr().String())

	defer join.Done()
	defer conn.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.Handler.ServeConn(ctx, conn)
}

func (s *Server) recover(err interface{}, addr string) {
	// Copied from https://golang.org/src/net/http/server.go
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	s.logf("panic serving %v: %v\n%s", addr, err, buf)
}

func (s *Server) logf(format string, args ...interface{}) {
	if s.ErrorLog == nil {
		log.Printf(format, args...)
	} else {
		s.ErrorLog.Printf(format, args...)
	}
}

func (s *Server) context() context.Context {
	ctx := s.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}
