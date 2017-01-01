package httpx

import (
	"bufio"
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
