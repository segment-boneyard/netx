package netx

import (
	"context"
	"errors"
	"io"
	"net"
	"time"
)

// ProtoReader is an interface implemented by protocols to figure out whether
// they recognize a data stream.
type ProtoReader interface {
	CanRead(io.Reader) bool
}

// Proto is an interface used to represent connection oriented protocols that
// can be plugged into a server to provide dynamic protocol discovery.
type Proto interface {
	Handler
	ProtoReader
}

// ProxyProto is an interface used to represent protocols that support
// proxying connections.
type ProxyProto interface {
	ProxyHandler
	ProtoReader
}

// TunnelProto is an interface used to represent protocols that support
// tunneling connections.
type TunnelProto interface {
	TunnelHandler
	ProtoReader
}

// ProtoMux is a connection handler that implement dynamic protocol discovery.
type ProtoMux struct {
	// Protocols is the list of supported protocols by the muxer.
	Protocols []Proto

	// ReadTimeout is the maximum amount of time the muxer will wait for the
	// first bytes to come.
	// Zero means no timeout.
	ReadTimeout time.Duration
}

// ServeConn satisifies the Handler interface.
//
// The method panics to report errors.
func (mux *ProtoMux) ServeConn(ctx context.Context, conn net.Conn) {
	readers := make([]ProtoReader, len(mux.Protocols))
	for i, p := range mux.Protocols {
		readers[i] = p
	}

	i, conn := guessProtocol(ctx, conn, mux.ReadTimeout, readers...)
	if i < 0 {
		panic(errUnsupportedProtocol)
	}

	mux.Protocols[i].ServeConn(ctx, conn)
}

// ProxyProtoMux is a proxy handler that implement dynamic protocol discovery.
type ProxyProtoMux struct {
	// Protocols is the list of supported protocols by the muxer.
	Protocols []ProxyProto

	// ReadTimeout is the maximum amount of time the muxer will wait for the
	// first bytes to come.
	// Zero means no timeout.
	ReadTimeout time.Duration
}

// ServeProxy satisfies the ProxyHandler interface.
//
// The method panics to report errors.
func (mux *ProxyProtoMux) ServeProxy(ctx context.Context, conn net.Conn, target net.Addr) {
	readers := make([]ProtoReader, len(mux.Protocols))
	for i, p := range mux.Protocols {
		readers[i] = p
	}

	i, conn := guessProtocol(ctx, conn, mux.ReadTimeout, readers...)
	if i < 0 {
		panic(errUnsupportedProtocol)
	}

	mux.Protocols[i].ServeProxy(ctx, conn, target)
}

// TunnelProtoMux is a tunnel handler that implement dynamic protocol discovery.
type TunnelProtoMux struct {
	// Protocols is the list of supported protocols by the muxer.
	Protocols []TunnelProto

	// ReadTimeout is the maximum amount of time the muxer will wait for the
	// first bytes to come.
	// Zero means no timeout.
	ReadTimeout time.Duration
}

// ServeTunnel satisfies the Tunnel
func (mux *TunnelProtoMux) ServeTunnel(ctx context.Context, from net.Conn, to net.Conn) {
	var ready1 <-chan struct{}
	var ready2 <-chan struct{}
	var cancel1 func()
	var cancel2 func()
	var err error

	if mux.ReadTimeout != 0 {
		newCtx, cancel := context.WithTimeout(ctx, mux.ReadTimeout)
		defer cancel()
		ctx = newCtx
	}

	// We're not sure which side of the connection is going to emit data first,
	// so we poll both connections and use the one that triggers first.
	if ready1, cancel1, err = PollRead(from); err != nil {
		panic(err)
	}
	defer cancel1()

	if ready2, cancel2, err = PollRead(to); err != nil {
		panic(err)
	}
	defer cancel2()

	readers := make([]ProtoReader, len(mux.Protocols))
	for i, p := range mux.Protocols {
		readers[i] = p
	}

	var i int
	select {
	case <-ready1:
		cancel2()
		i, from = guessProtocol(ctx, from, mux.ReadTimeout, readers...)
	case <-ready2:
		cancel1()
		i, to = guessProtocol(ctx, to, mux.ReadTimeout, readers...)
	case <-ctx.Done():
		return
	}

	if i < 0 {
		panic(errUnsupportedProtocol)
	}

	mux.Protocols[i].ServeTunnel(ctx, from, to)
}

func guessProtocol(ctx context.Context, conn net.Conn, timeout time.Duration, protos ...ProtoReader) (int, net.Conn) {
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-done:
		case <-ctx.Done():
			conn.Close()
		}
	}()

	if timeout != 0 {
		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			panic(err)
		}
	}

	tr := &teeReader{
		r: conn,
		b: make([]byte, 0, 1024),
	}

	for i, proto := range protos {
		if proto.CanRead(tr) {
			return i, &protoConn{conn, tr.bytes()}
		}
		tr.reset()
	}

	return -1, &protoConn{conn, tr.bytes()}
}

// teeReader is an io.Reader which records all data it reads, then can be reset
// to replay them.
type teeReader struct {
	r io.Reader
	b []byte
	i int
}

func (t *teeReader) reset() {
	t.i = 0
}

func (t *teeReader) bytes() []byte {
	return t.b
}

func (t *teeReader) Read(b []byte) (n int, err error) {
	if t.i < len(t.b) {
		n1 := len(t.b) - t.i
		n2 := len(b)

		if n2 > n1 {
			n2 = n1
		}

		copy(b, t.b[t.i:t.i+n2])
		t.i += n2
		n = n2
		return
	}

	if n, err = t.r.Read(b); n > 0 {
		t.b = append(t.b, b[:n]...)
		t.i += n
	}

	return
}

// protoConn is a net.Conn which is preloaded with some data that will be read
// by calls to Read before consuming from the underlying network connection.
type protoConn struct {
	net.Conn
	head []byte
}

func (c *protoConn) Read(b []byte) (n int, err error) {
	if n = len(c.head); n != 0 {
		if n > len(b) {
			n = len(b)
		}
		copy(b, c.head[:n])
		if c.head = c.head[n:]; len(c.head) == 0 {
			c.head = nil // release the buffer
		}
		b = b[n:]
		return
	}
	return c.Conn.Read(b)
}

var (
	errUnsupportedProtocol = errors.New("unsupported protocol")
)
