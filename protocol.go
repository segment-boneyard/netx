package netx

import (
	"context"
	"errors"
	"net"
	"time"
)

// ProtoReader is an interface implemented by protocols to figure out whether
// they recognize a blob of data.
type ProtoReader interface {
	CanRead([]byte) bool
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
	var b []byte
	var err error

	if conn, b, err = peekContextTimeout(ctx, conn, mux.ReadTimeout); err != nil {
		return // timeout or the connection was closed
	}

	for _, proto := range mux.Protocols {
		if proto.CanRead(b) {
			proto.ServeConn(ctx, conn)
			return
		}
	}

	panic(errUnsupportedProtocol)
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
	var b []byte
	var err error

	if conn, b, err = peekContextTimeout(ctx, conn, mux.ReadTimeout); err != nil {
		return // timeout or the connection was closed
	}

	for _, proto := range mux.Protocols {
		if proto.CanRead(b) {
			proto.ServeProxy(ctx, conn, target)
			return
		}
	}

	panic(errUnsupportedProtocol)
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
		ctx, _ = context.WithTimeout(ctx, mux.ReadTimeout)
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

	var b []byte
	select {
	case <-ready1:
		cancel2()
		from, b, err = peek(from)
	case <-ready2:
		cancel1()
		to, b, err = peek(to)
	case <-ctx.Done():
		return
	}

	if err != nil {
		return // one of the connections were closed
	}

	for _, proto := range mux.Protocols {
		if proto.CanRead(b) {
			proto.ServeTunnel(ctx, from, to)
			return
		}
	}

	panic(errUnsupportedProtocol)
}

func peekContextTimeout(ctx context.Context, conn net.Conn, timeout time.Duration) (net.Conn, []byte, error) {
	if timeout != 0 {
		ctx, _ = context.WithTimeout(ctx, timeout)
	}
	return peekContext(ctx, conn)
}

func peekContext(ctx context.Context, conn net.Conn) (net.Conn, []byte, error) {
	var ready <-chan struct{}
	var cancel func()
	var err error

	if ready, cancel, err = PollRead(conn); err != nil {
		return conn, nil, err
	}
	defer cancel()

	select {
	case <-ready:
	case <-ctx.Done():
		return conn, nil, ctx.Err()
	}

	return peek(conn)
}

func peek(conn net.Conn) (net.Conn, []byte, error) {
	b := make([]byte, 512)

	n, err := conn.Read(b)
	if err != nil {
		return conn, nil, err
	}

	return &protoConn{conn, b[:n]}, b[:n], nil
}

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
		c.head = c.head[n:]
		return
	}
	return c.Conn.Read(b)
}

var (
	errUnsupportedProtocol = errors.New("unsupported protocol")
)
