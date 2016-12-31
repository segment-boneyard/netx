package httpx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

	baseHeader := http.Header{
		"Content-Type": {"application/octet-stream"},
		"Server":       {s.ServerName},
	}
	if idleTimeout := s.IdleTimeout; idleTimeout != 0 {
		baseHeader.Set("Connection", "Keep-Alive")
		baseHeader.Set("Keep-Alive", fmt.Sprintf("timeout=%d", int(idleTimeout/time.Second)))
	}

	sc := newServerConn(conn)
	defer sc.Close()

	res := &responseWriter{
		header:  make(http.Header, 10),
		conn:    sc,
		timeout: s.WriteTimeout,
	}
	copyHeader(res.header, baseHeader)

	for {
		var req *http.Request
		var err error
		var closed bool

		if err = sc.waitReadyRead(ctx, s.IdleTimeout); err != nil {
			return
		}
		if req, err = sc.readRequest(ctx, maxHeaderBytes, s.ReadTimeout); err != nil {
			return
		}
		res.req = req

		if closed = req.Close; closed {
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
		if closed || req.Close {
			return
		}

		netx.Copy(ioutil.Discard, req.Body)
		req.Body.Close()

		res.reset(baseHeader)
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

func (s *Server) serveHTTP(w http.ResponseWriter, req *http.Request, conn net.Conn) {
	defer func() {
		res := w.(*responseWriter)
		err := recover()

		if err != nil {
			netx.Recover(err, conn, s.ErrorLog)

			// If the header wasn't written yet when the error occurred we can
			// attempt to keep using the connection, otherwise we abort to
			// notify the client that something went wrong.
			if res.status == 0 {
				res.WriteHeader(http.StatusInternalServerError)
			} else {
				req.Close = true
				return
			}
		}

		res.close()
		res.Flush()
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

	handler.ServeHTTP(w, req)
}

// serverConn is a net.Conn that embeds a I/O buffers and a connReader, this is
// mainly used as an optimization to reduce the number of dynamic memory
// allocations.
type serverConn struct {
	c connReader
	f *os.File
	bufio.Reader
	bufio.Writer
}

func newServerConn(conn net.Conn) *serverConn {
	c := &serverConn{c: connReader{Conn: conn, limit: -1}}
	c.Reader = *bufio.NewReader(&c.c)
	c.Writer = *bufio.NewWriter(conn)
	if f, ok := conn.(netx.File); ok {
		c.f, _ = f.File()
	}
	return c
}

func (conn *serverConn) LocalAddr() net.Addr                { return conn.c.LocalAddr() }
func (conn *serverConn) RemoteAddr() net.Addr               { return conn.c.RemoteAddr() }
func (conn *serverConn) SetDeadline(t time.Time) error      { return conn.c.SetDeadline(t) }
func (conn *serverConn) SetReadDeadline(t time.Time) error  { return conn.c.SetReadDeadline(t) }
func (conn *serverConn) SetWriteDeadline(t time.Time) error { return conn.c.SetWriteDeadline(t) }

func (conn *serverConn) Close() error {
	conn.closeFile()
	return conn.c.Close()
}

func (conn *serverConn) closeFile() {
	if conn.f != nil {
		conn.f.Close()
	}
}

func (conn *serverConn) waitReadyRead(ctx context.Context, timeout time.Duration) (err error) {
	if conn.f != nil {
		err = waitRead(ctx, conn.f, timeout)
	} else {
		err = pollRead(ctx, conn, &conn.Reader, timeout)
	}
	return
}

func (conn *serverConn) readRequest(ctx context.Context, maxHeaderBytes int, timeout time.Duration) (req *http.Request, err error) {
	// Limit the size of the request header, if readRequest attempts to read
	// more than maxHeaderBytes it will get io.EOF.
	conn.c.limit = maxHeaderBytes

	if timeout != 0 {
		conn.SetReadDeadline(time.Now().Add(timeout))
	} else {
		conn.SetReadDeadline(time.Time{})
	}

	if req, err = http.ReadRequest(&conn.Reader); err != nil {
		return
	}

	ctx = context.WithValue(ctx, http.LocalAddrContextKey, conn.LocalAddr())
	req = req.WithContext(ctx)
	req.RemoteAddr = conn.RemoteAddr().String()

	// Remove the "close" and "keep-alive" Connection header values, these values
	// are automatically handled by the server and reported in req.Close.
	if h, ok := req.Header["Connection"]; ok {
		req.Header["Connection"] = headerValuesRemoveTokens(h, "close", "keep-alive")
	}

	// Drop the size limit on the connection reader to let the request body
	// go through.
	conn.c.limit = -1
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
	chunked bool
	cw      chunkWriter
}

// Hijack satisfies the http.Hijacker interface.
func (res *responseWriter) Hijack() (conn net.Conn, rw *bufio.ReadWriter, err error) {
	if res.err != nil {
		err = res.err
		return
	}

	if res.chunked {
		if err = res.cw.Flush(); err != nil {
			res.err = err
			return
		}
	}

	conn, rw = res.conn.c.Conn, bufio.NewReadWriter(&res.conn.Reader, &res.conn.Writer)
	res.conn.closeFile()
	res.conn = nil
	res.err = errHijacked

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

	// The chunkWriter's buffer is unused for now, we'll use it to write the
	// status line and avoid a couple of memory allocations (because byte
	// slices sent to the bufio.Writer will be seen as escaping to the
	// underlying io.Writer).
	var b = res.cw.b[:0]
	var c = res.conn
	var h = res.header

	if timeout := res.timeout; timeout != 0 {
		c.SetWriteDeadline(time.Now().Add(timeout))
	}

	if _, hasLen := h["Content-Length"]; !hasLen {
		h.Set("Transfer-Encoding", "chunked")
		res.chunked = true
		res.cw.w = res.conn
		res.cw.n = 0
	}

	h.Set("Date", now().Format(time.RFC1123))

	b = append(b, res.req.Proto...)
	b = append(b, ' ')
	b = strconv.AppendInt(b, int64(status), 10)
	b = append(b, ' ')
	b = append(b, http.StatusText(status)...)
	b = append(b, '\r', '\n')

	if _, err := c.Write(b); err != nil {
		res.err = err
		return
	}
	if err := h.Write(c); err != nil {
		res.err = err
		return
	}
	if _, err := c.WriteString("\r\n"); err != nil {
		res.err = err
		return
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

	if res.chunked {
		n, err = res.cw.Write(b)
	} else {
		n, err = res.conn.Write(b)
	}

	res.err = err
	return
}

// Flush satsifies the http.Flusher interface.
func (res *responseWriter) Flush() {
	if res.conn != nil && res.err == nil {
		res.WriteHeader(0)

		if res.err == nil {
			if res.chunked {
				if res.err = res.cw.Flush(); res.err != nil {
					return
				}
			}
			res.err = res.conn.Flush()
		}
	}
}

func (res *responseWriter) close() {
	if res.chunked {
		res.WriteHeader(0)

		if res.err == nil {
			res.err = res.cw.Close()
		}
	}
}

func (res *responseWriter) reset(baseHeader http.Header) {
	res.chunked = false
	res.cw.w = nil
	res.cw.n = 0
	res.req = nil
	res.header = make(http.Header, 10)
	copyHeader(res.header, baseHeader)
}

// chunkWriter provides the implementation of an HTTP writer that outputs a
// response body using the chunked transfer encoding.
type chunkWriter struct {
	w io.Writer // writer to which data are flushed
	n int       // offset in of the last byte in b
	a [8]byte   // buffer used for writing the chunk size
	b [512]byte // buffer used to aggregate small chunks
}

func (res *chunkWriter) Write(b []byte) (n int, err error) {
	for len(b) != 0 {
		n1 := len(b)
		n2 := len(res.b) - res.n

		if n1 >= n2 {
			if res.n == 0 {
				// Nothing is buffered and we have a large chunk already, bypass
				// the chunkWriter's buffer and directly output to its writer.
				return res.writeChunk(b)
			}
			n1 = n2
		}

		copy(res.b[res.n:], b[:n1])
		res.n += n1
		n += n1

		if b = b[n1:]; len(b) != 0 {
			if err = res.Flush(); err != nil {
				break
			}
		}
	}
	return
}

func (res *chunkWriter) Close() (err error) {
	if err = res.Flush(); err == nil {
		_, err = res.w.Write(append(res.a[:0], "0\r\n\r\n"...))
	}
	return
}

func (res *chunkWriter) Flush() (err error) {
	var n int

	if n, err = res.writeChunk(res.b[:res.n]); n > 0 {
		if n == res.n {
			res.n = 0
		} else {
			// Not all buffered data could be flushed, moving the bytes to the
			// front of the chunkWriter's buffer.
			copy(res.b[:], res.b[n:res.n])
			res.n -= n
		}
	}

	return
}

func (res *chunkWriter) writeChunk(b []byte) (n int, err error) {
	if len(b) == 0 {
		// Don't write empty chunks, they would be misinterpreted as the end of
		// the stream.
		return
	}

	a := append(strconv.AppendInt(res.a[:0], int64(len(b)), 16), '\r', '\n')

	if _, err = res.w.Write(a); err != nil {
		return
	}
	if n, err = res.w.Write(b); err != nil {
		return
	}
	_, err = res.w.Write(a[len(a)-2:]) // CRLF
	return
}

var (
	timezone    = time.FixedZone("GMT", 0)
	errHijacked = errors.New("the HTTP connection has already been hijacked")
)

func now() time.Time {
	return time.Now().In(timezone)
}
