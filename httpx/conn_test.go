package httpx

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/segmentio/netx/httpx/httpxtest"
)

func TestConnTransportDefault(t *testing.T) {
	httpxtest.TestTransport(t, func() http.RoundTripper {
		return &ConnTransport{}
	})
}

func TestConnTransportConfigured(t *testing.T) {
	httpxtest.TestTransport(t, func() http.RoundTripper {
		return &ConnTransport{
			Buffer: &bufio.ReadWriter{
				Reader: bufio.NewReader(nil),
				Writer: bufio.NewWriter(nil),
			},
			ResponseHeaderTimeout:  100 * time.Millisecond,
			MaxResponseHeaderBytes: 65536,
		}
	})
}

func TestConnReader(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	ok := false
	cr := &connReader{
		Conn:   c1,
		limit:  10,
		cancel: func() { ok = true },
	}

	go c2.Write([]byte("Hello World!"))
	var b [16]byte

	n, err := cr.Read(b[:])
	if err != nil {
		t.Error(err)
		return
	}

	if n > 10 {
		t.Error("too many bytes read from c1:", n)
		return
	}

	if _, err := cr.Read(b[:]); err != io.EOF {
		t.Error("expected io.EOF but got", err)
		return
	}
}
