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
//
// The server becomes the owner of the listener which will be closed by the time
// the Serve method returns.
func (s *Server) Serve(lstn net.Listener) error {
	join := &sync.WaitGroup{}
	defer join.Wait()
	defer lstn.Close()

	ctx := s.Context
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := ctx.Done()
	errs := make(chan error)
	conns := make(chan net.Conn)

	join.Add(1)
	go s.accept(ctx, lstn, conns, errs, join)

	for conns != nil || errs != nil {
		select {
		case <-done:
			lstn.Close()
			done = nil

		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			return err

		case conn, ok := <-conns:
			if !ok {
				conns = nil
				continue
			}
			join.Add(1)
			go s.serve(ctx, conn, join)
		}
	}

	return nil
}

func (s *Server) accept(ctx context.Context, lstn net.Listener, conns chan<- net.Conn, errs chan<- error, join *sync.WaitGroup) {
	defer join.Done()
	defer close(errs)
	defer close(conns)

	const maxBackoff = 1 * time.Second
	for {
		var conn net.Conn
		var err error

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
			case <-ctx.Done():
				// Don't report errors when the server stopped because its
				// context was canceled.
			default:
				errs <- err
			}
			return
		}

		conns <- conn
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
