package httpx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/netx"
)

const (
	// DefaultMaxHeaderBytes is the default value used for limiting the size of
	// HTTP request headers.
	DefaultMaxHeaderBytes = 1048576
)

// A Server implements the netx.Handler interface, it provides the handling of
// HTTP requests from a net.Conn, graceful shutdowns...
type Server struct {
	// Handler is called by the server for each HTTP request it received.
	Handler http.Handler

	// Upgrader is called by the server when an HTTP upgrade is detected.
	Upgrader http.Handler

	// IdleTimeout is the maximum amount of time the server waits on an inactive
	// connection before closing it.
	// Zero means no timeout.
	IdleTimeout time.Duration

	// ReadTimeout is the maximum amount of time the server waits for a request
	// to be fully read.
	// Zero means no timeout.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum amount of time the server gives for responses
	// to be sent.
	// Zero means no timeout.
	WriteTimeout time.Duration

	// MaxHeaderBytes controls the maximum number of bytes the will read parsing
	// the request header's keys and values, including the request line. It does
	// not limit the size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int

	// ErrorLog specifies an optional logger for errors that occur when
	// attempting to proxy the request. If nil, logging goes to os.Stderr via
	// the log package's standard logger.
	ErrorLog *log.Logger

	// ServerName is the name of the server, returned in the "Server" response
	// header field.
	ServerName string
}

// ServeConn satisfies the netx.Handler interface.
func (s *Server) ServeConn(ctx context.Context, conn net.Conn) {
	maxHeaderBytes := s.MaxHeaderBytes
	if maxHeaderBytes == 0 {
		maxHeaderBytes = DefaultMaxHeaderBytes
	}

	baseHeader := http.Header{"Server": {s.ServerName}}
	if idleTimeout := s.IdleTimeout; idleTimeout != 0 {
		baseHeader.Set("Connection", "Keep-Alive")
		baseHeader.Set("Keep-Alive", fmt.Sprintf("timeout=%d", int(idleTimeout/time.Second)))
	}

	srv := newServerConn(conn)
	res := &responseWriter{
		header:  http.Header{},
		conn:    srv,
		timeout: s.WriteTimeout,
	}
	copyHeader(res.header, baseHeader)

	for close := false; !close; {
		var req *http.Request
		var err error

		if err = srv.waitReadyRead(ctx, s.IdleTimeout); err != nil {
			return
		}

		if req, err = srv.readRequest(ctx, maxHeaderBytes, s.ReadTimeout); err != nil {
			return
		}
		res.req = req

		if close = req.Close; close {
			if req.ProtoAtLeast(1, 1) {
				res.header.Add("Connection", "close")
			}
		} else {
			if protoEqual(req, 1, 0) {
				res.header.Add("Connection", "keep-alive")
			}
		}

		s.serveHTTP(res, req, conn)

		if res.conn == nil { // hijacked
			return
		}

		if res.err != nil { // probably lost the connection
			return
		}

		res.header = http.Header{}
		copyHeader(res.header, baseHeader)
	}
}

// ServeProxy satisfies the netx.ProxyHandler interface, it is used to support
// transparent HTTP proxies, it rewrites the request to take into account the
// fact that it was received on an intercepted connection and that the client
// wasn't aware that it was being proxied.
func (s *Server) ServeProxy(ctx context.Context, conn net.Conn, target net.Addr) {
	handler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		scheme, _ := splitProtoAddr(req.Host)

		// If the Host had no scheme we're propbably in a transparent proxy and
		// the client didn't know it had to place the full URL in the header.
		// We attempt to guess the protocol from the network connection itself.
		if len(scheme) == 0 {
			if conn.LocalAddr().Network() == "tls" {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}

		// Rewrite the URL to which the request will be forwarded.
		req.URL.Scheme = scheme
		req.URL.Host = target.String()

		// Fallback to the orignal server's handler.
		s.serveHTTP(res, req, conn)
	})
	server := *s
	server.Upgrader = handler
	server.Handler = handler
	server.ServeConn(ctx, conn)
}

func (s *Server) serveHTTP(res http.ResponseWriter, req *http.Request, conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			netx.Recover(err, conn, s.ErrorLog)
			res.WriteHeader(http.StatusInternalServerError)
		} else {
			res.WriteHeader(0)
		}
		res.(*responseWriter).Flush()
	}()
	handler := s.Handler
	upgrade := connectionUpgrade(req.Header)

	switch {
	case len(req.Header["Expect"]) != 0:
		handler = StatusHandler(http.StatusExpectationFailed)

	case len(upgrade) != 0:
		if s.Upgrader == nil {
			handler = StatusHandler(http.StatusNotImplemented)
		} else {
			handler = s.Upgrader
		}
	}

	handler.ServeHTTP(res, req)
}

// serverConn is a net.Conn that embeds a I/O buffers and a connReader, this is
// mainly used as an optimization to reduce the number of dynamic memory
// allocations.
type serverConn struct {
	connReader
	r bufio.Reader
	w bufio.Writer
	f *os.File
}

func newServerConn(conn net.Conn) *serverConn {
	c := &serverConn{connReader: connReader{Conn: conn}}
	c.r = *bufio.NewReader(c)
	c.w = *bufio.NewWriter(c)
	if f, ok := conn.(netx.File); ok {
		c.f, _ = f.File()
	}
	return c
}

func (c *serverConn) waitReadyRead(ctx context.Context, timeout time.Duration) (err error) {
	if c.f == nil {
		err = pollRead(ctx, c, &c.r, timeout)
	} else {
		err = waitRead(ctx, c.f, timeout)
	}
	return
}

func (c *serverConn) readRequest(ctx context.Context, maxHeaderBytes int, timeout time.Duration) (req *http.Request, err error) {
	// Limit the size of the request header, if readRequest attempts to read
	// more than maxHeaderBytes it will get io.EOF.
	c.limit = maxHeaderBytes

	if timeout != 0 {
		c.SetReadDeadline(time.Now().Add(timeout))
	} else {
		c.SetReadDeadline(time.Time{})
	}

	if req, err = http.ReadRequest(&c.r); err != nil {
		return
	}

	ctx = context.WithValue(ctx, http.LocalAddrContextKey, c.LocalAddr())
	req = req.WithContext(ctx)
	req.RemoteAddr = c.RemoteAddr().String()

	// Remove the "close" and "keep-alive" Connection header values, these values
	// are automatically handled by the server and reported in req.Close.
	if h, ok := req.Header["Connection"]; ok {
		req.Header["Connection"] = headerValuesRemoveTokens(h, "close", "keep-alive")
	}

	// Drop the size limit on the connection reader to let the request body
	// go through.
	c.limit = -1
	return
}

// responseWriter is an implementation of the http.ResponseWriter interface.
//
// Instances of responseWriter provide most of the features exposed in the
// standard library, however there are a couple differences:
// - There is no support for HTTP trailers.
// - No automatic detection of the content type is done, this doesn't work
// anyway if a content encoding is managed by a request handler.
type responseWriter struct {
	status  int
	header  http.Header
	conn    *serverConn
	req     *http.Request
	timeout time.Duration
	err     error
}

// Hijack satisfies the http.Hijacker interface.
func (res *responseWriter) Hijack() (conn net.Conn, rw *bufio.ReadWriter, err error) {
	if res.conn == nil {
		err = errHijacked
		return
	}

	conn, rw, err = res.conn, bufio.NewReadWriter(&res.conn.r, &res.conn.w), res.err
	res.conn = nil

	// Cancel all deadlines on the connection before returning it.
	conn.SetDeadline(time.Time{})
	return
}

// Header satisfies the http.ResponseWriter interface.
func (res *responseWriter) Header() http.Header {
	return res.header
}

// WriteHeader satisfies the http.ResponseWriter interface.
func (res *responseWriter) WriteHeader(status int) {
	if res.status != 0 {
		return
	}

	if status == 0 {
		status = http.StatusOK
	}

	res.status = status

	w := &res.conn.w
	h := res.header
	p := res.req.Proto
	t := res.timeout
	b := [32]byte{}

	w.WriteString(p)
	w.WriteByte(' ')
	w.WriteString(string(strconv.AppendInt(b[:0], int64(status), 10)))
	w.WriteByte(' ')
	w.WriteString(http.StatusText(status))
	w.WriteString("\r\n")
	h.Write(w)
	_, res.err = w.WriteString("\r\n")

	if t != 0 {
		res.conn.SetWriteDeadline(time.Now().Add(t))
	}
}

// Write satisfies the io.Writer and http.ResponseWriter interfaces.
func (res *responseWriter) Write(b []byte) (n int, err error) {
	if res.conn == nil {
		err = errHijacked
		return
	}

	if err = res.err; err != nil {
		return
	}

	res.WriteHeader(0)
	n, err = res.conn.w.Write(b)
	res.err = err
	return
}

// Flush satsifies the http.Flusher interface.
func (res *responseWriter) Flush() {
	if res.conn != nil && res.err == nil {
		res.WriteHeader(0)
		res.err = res.conn.w.Flush()
	}
}

var (
	errHijacked = errors.New("the HTTP connection has already been hijacked")
)
