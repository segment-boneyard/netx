package httpx

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"
)

// ConnTransport is a http.RoundTripper that works on a pre-established network
// connection.
type ConnTransport struct {
	// Conn is the connection to use to send requests and receive responses.
	Conn net.Conn

	// Buffer may be set to a bufio.ReadWriter which will be used to buffer all
	// I/O done on the connection.
	Buffer *bufio.ReadWriter

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

// RoundTrip satisfies the http.RoundTripper interface.
func (t *ConnTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	var c = &connReader{Conn: t.Conn, limit: -1}
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
		c.limit = DefaultMaxHeaderBytes
	case limit > 0:
		c.limit = limit
	}

	if timeout := t.ResponseHeaderTimeout; timeout != 0 {
		c.SetReadDeadline(time.Now().Add(timeout))
	}
	if res, err = http.ReadResponse(r, req); err != nil {
		return
	}

	c.limit = -1
	c.SetReadDeadline(time.Time{})
	return
}

// connReader is a net.Conn wrappers used by the HTTP server to limit the size
// of the request header.
type connReader struct {
	net.Conn
	limit int
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

	return
}
