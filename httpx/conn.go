package httpx

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/segmentio/netx"
)

// ConnTransport is a http.RoundTripper that works on a pre-established network
// connection.
type ConnTransport struct {
	// Conn is the connection to use to send requests and receive responses.
	Conn net.Conn

	// Buffer may be set to a bufio.ReadWriter which will be used to buffer all
	// I/O done on the connection.
	Buffer *bufio.ReadWriter

	// DialContext is used to open a connection when Conn is set to nil.
	// If the function is nil the transport uses a default dialer.
	DialContext func(context.Context, string, string) (net.Conn, error)

	// ResponseHeaderTimeout, if non-zero, specifies the amount of time to wait
	// for a server's response headers after fully writing the request (including
	// its body, if any). This time does not include the time to read the response
	// body.
	ResponseHeaderTimeout time.Duration

	// MaxResponseHeaderBytes specifies a limit on how many response bytes are
	// allowed in the server's response header.
	//
	// Zero means to use a default limit.
	MaxResponseHeaderBytes int
}

// the default dialer used by ConnTransport when neither Conn nor DialContext is
// set.
var dialer net.Dialer

// RoundTrip satisfies the http.RoundTripper interface.
func (t *ConnTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	var ctx = req.Context()
	var conn net.Conn
	var dial func(context.Context, string, string) (net.Conn, error)

	if conn = t.Conn; conn == nil {
		if dial = t.DialContext; dial == nil {
			dial = dialer.DialContext
		}
		if conn, err = dial(ctx, "tcp", req.Host); err != nil {
			return
		}
	}

	var c = &connReader{Conn: conn, limit: -1}
	var b = t.Buffer
	var r *bufio.Reader
	var w *bufio.Writer

	if b != nil && b.Reader != nil {
		r = b.Reader
		r.Reset(c)
	} else {
		r = bufio.NewReader(c)
	}

	if b != nil && b.Writer != nil {
		w = b.Writer
		w.Reset(c)
	} else {
		w = bufio.NewWriter(c)
	}

	if err = req.Write(w); err != nil {
		return
	}
	if err = w.Flush(); err != nil {
		return
	}

	switch limit := t.MaxResponseHeaderBytes; {
	case limit == 0:
		c.limit = http.DefaultMaxHeaderBytes
	case limit > 0:
		c.limit = limit
	}

	if timeout := t.ResponseHeaderTimeout; timeout != 0 {
		c.SetReadDeadline(time.Now().Add(timeout))
	}
	if res, err = http.ReadResponse(r, req); err != nil {
		return
	}

	if dial != nil {
		res.Body = struct {
			io.Reader
			io.Closer
		}{
			Reader: res.Body,
			Closer: conn,
		}
	}

	c.limit = -1
	c.SetReadDeadline(time.Time{})
	return
}

// connReader is a net.Conn wrappers used by the HTTP server to limit the size
// of the request header.
//
// A cancel function can also be set on the reader, it is expected to be used to
// cancel the associated request context to notify the handlers that a client is
// gone and the request can be aborted.
type connReader struct {
	net.Conn
	limit  int
	cancel context.CancelFunc
}

// Read satsifies the io.Reader interface.
func (c *connReader) Read(b []byte) (n int, err error) {
	if c.limit == 0 {
		err = io.EOF
		return
	}

	n1 := len(b)
	n2 := c.limit

	if n2 > 0 && n1 > n2 {
		n1 = n2
	}

	if n, err = c.Conn.Read(b[:n1]); n > 0 && n2 > 0 {
		c.limit -= n
	}

	if err != nil && !netx.IsTemporary(err) {
		c.cancel()
	}

	return
}
