package netx

import (
	"bufio"
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
		if _, err := conn.Write([]byte("/rev Hello World!\n")); err != nil {
			t.Error(err)
			return
		}

		if n, err := io.ReadFull(conn, b[:13]); err != nil {
			t.Error(n, err, string(b[:n]))
			return
		}

		if s := string(b[:13]); s != "!dlroW olleH\n" {
			t.Error(s)
			return
		}
	})
}

// protoEcho implements the Proto* interfaces on top of a Echo handler.
type protoEcho struct{ Echo }

func (p *protoEcho) CanRead(r io.Reader) bool                     { return true }
func (p *protoEcho) ServeConn(ctx context.Context, conn net.Conn) { p.Echo.ServeConn(ctx, conn) }

// protoEchoRev implements the Proto* interfaces, it's similar to a Echo handler
// but reverses data chunks before returning them.
type protoEchoRev struct{}

func (p *protoEchoRev) CanRead(r io.Reader) bool {
	var b [5]byte

	if _, err := io.ReadFull(r, b[:]); err != nil {
		return false
	}

	return string(b[:]) == "/rev "
}

func (p *protoEchoRev) ServeConn(ctx context.Context, conn net.Conn) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return
		}

		if !bytes.HasPrefix(line, []byte("/rev ")) {
			return
		}

		line = line[5:]

		for i, j := 0, len(line)-2; i < j; {
			line[i], line[j] = line[j], line[i]
			i++
			j--
		}
		if _, err = conn.Write(line); err != nil {
			return
		}
	}
}
