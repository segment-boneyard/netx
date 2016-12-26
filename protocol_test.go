package netx

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestProtoMux(t *testing.T) {
	testProtoMux(t, &ProtoMux{
		Protocols: []Proto{
			&protoEchoRev{},
			&protoEcho{},
		},
		ReadTimeout: 100 * time.Millisecond,
	})
}

func TestProxyProtoMux(t *testing.T) {
	net0, addr0, close0 := listenAndServe(&ProtoMux{
		Protocols: []Proto{
			&protoEchoRev{},
			&protoEcho{},
		},
		ReadTimeout: 100 * time.Millisecond,
	})
	defer close0()

	testProtoMux(t, &Proxy{
		Network: net0,
		Address: addr0,
		Handler: &ProxyProtoMux{
			Protocols:   []ProxyProto{&Tunnel{Handler: &Forwarder{}}},
			ReadTimeout: 100 * time.Millisecond,
		},
	})
}

func TestTunnelProtoMux(t *testing.T) {
	net0, addr0, close0 := listenAndServe(&ProtoMux{
		Protocols: []Proto{
			&protoEchoRev{},
			&protoEcho{},
		},
		ReadTimeout: 100 * time.Millisecond,
	})
	defer close0()

	testProtoMux(t, &Proxy{
		Network: net0,
		Address: addr0,
		Handler: &Tunnel{
			Handler: &TunnelProtoMux{
				Protocols:   []TunnelProto{&Forwarder{}},
				ReadTimeout: 100 * time.Millisecond,
			},
		},
	})
}

func testProtoMux(t *testing.T, handler Handler) {
	net0, addr0, close0 := listenAndServe(handler)
	defer close0()

	t.Run("route-1", func(t *testing.T) {
		conn, err := net.Dial(net0, addr0)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		var b [512]byte

		// test the protoEcho route
		if _, err := conn.Write([]byte("Hello World!")); err != nil {
			t.Error(err)
			return
		}

		if _, err := io.ReadFull(conn, b[:12]); err != nil {
			t.Error(err)
			return
		}

		if s := string(b[:12]); s != "Hello World!" {
			t.Error(s)
			return
		}
	})

	t.Run("route-2", func(t *testing.T) {
		conn, err := net.Dial(net0, addr0)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		var b [512]byte

		// test the protoEchoRev route
		if _, err := conn.Write([]byte("/rev Hello World!")); err != nil {
			t.Error(err)
			return
		}

		if _, err := io.ReadFull(conn, b[:12]); err != nil {
			t.Error(err)
			return
		}

		if s := string(b[:12]); s != "!dlroW olleH" {
			t.Error(s)
			return
		}
	})
}

// protoEcho implements the Proto* interfaces on top of a Echo handler.
type protoEcho struct{ Echo }

func (p *protoEcho) CanRead(b []byte) bool                        { return true }
func (p *protoEcho) ServeConn(ctx context.Context, conn net.Conn) { p.Echo.ServeConn(ctx, conn) }

// protoEchoRev implements the Proto* interfaces, it's similar to a Echo handler
// but reverses data chunks before returning them.
type protoEchoRev struct{}

func (p *protoEchoRev) CanRead(b []byte) bool {
	return bytes.HasPrefix(b, []byte("/rev "))
}

func (p *protoEchoRev) ServeConn(ctx context.Context, conn net.Conn) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	var b [512]byte
	for {
		n, err := conn.Read(b[:])
		if err != nil {
			return
		}
		for i, j := 0, n-1; i < j; {
			b[i], b[j] = b[j], b[i]
			i++
			j--
		}
		if _, err = conn.Write(b[:n-4]); err != nil {
			return
		}
	}
}
