package netx

import (
	"io"
	"net"
	"testing"
)

func TestTunnel(t *testing.T) {
	tests := []struct {
		name   string
		tunnel TunnelHandler
	}{
		{
			name:   "TunnelRaw",
			tunnel: TunnelRaw,
		},
		{
			name:   "TunnelLine",
			tunnel: TunnelLine,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr1, close1 := listenAndServe(Echo)
			defer close1()

			addr2, close2 := listenAndServe(&Proxy{
				Addr: addr1,
				Handler: &Tunnel{
					Handler: test.tunnel,
				},
			})
			defer close2()

			conn, err := net.Dial(addr2.Network(), addr2.String())
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()

			if _, err := io.WriteString(conn, "Hello World!\r\n"); err != nil {
				t.Error(err)
				return
			}

			b := [14]byte{}

			if _, err := io.ReadFull(conn, b[:]); err != nil {
				t.Error(err)
				return
			}

			if s := string(b[:]); s != "Hello World!\r\n" {
				t.Error(s)
				return
			}
		})
	}
}
