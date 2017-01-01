package netx

import (
	"context"
	"io"
	"net"
)

// ProxyHandler is an interface that must be implemented by types that intend to
// proxy connections.
//
// The ServeProxy method is called by a Proxy when it receives a new connection.
// It is similar to the ServeConn method of the Handler interface but receives
// an extra target argument representing the original address that the
// intercepted connection intended to reach.
type ProxyHandler interface {
	ServeProxy(ctx context.Context, conn net.Conn, target net.Addr)
}

// ProxyHandlerFunc makes it possible for simple function types to be used as
// connection proxies.
type ProxyHandlerFunc func(context.Context, net.Conn, net.Addr)

// ServeProxy calls f.
func (f ProxyHandlerFunc) ServeProxy(ctx context.Context, conn net.Conn, target net.Addr) {
	f(ctx, conn, target)
}

// A Proxy is a connection handler that forwards its connections to a proxy
// handler.
type Proxy struct {
	// Network and Address represent the target to which the proxy is forwarding
	// connections.
	Network string
	Address string

	// Handler is the proxy handler to which connetions are forwarded to.
	Handler ProxyHandler
}

// CanRead satisfies the ProtoReader interface, always returns true. This means
// that a proxy can be used as a fallback protocol in a ProtoMux.
func (p *Proxy) CanRead(r io.Reader) bool {
	return true
}

// ServeConn satsifies the Handler interface.
func (p *Proxy) ServeConn(ctx context.Context, conn net.Conn) {
	p.Handler.ServeProxy(ctx, conn, &NetAddr{
		Net:  p.Network,
		Addr: p.Address,
	})
}

// A TransparentProxy is a connection handler for intercepted connections.
//
// A proper usage of this proxy requires some iptables rules to redirect TCP
// connections to to the listener its attached to.
type TransparentProxy struct {
	// Handler is called by the proxy when it receives a connection that can be
	// proxied.
	//
	// Calling ServeConn on the proxy will panic if this field is nil.
	Handler ProxyHandler
}

// ServeConn satisfies the Handler interface.
//
// The method panics to report errors.
func (p *TransparentProxy) ServeConn(ctx context.Context, conn net.Conn) {
	target, err := OriginalTargetAddr(conn)
	if err != nil {
		panic(err)
	}
	p.Handler.ServeProxy(ctx, conn, target)
}

// OriginalTargetAddr returns the original address that an intercepted
// connection intended to reach.
//
// Note that this feature is only available for TCP connections on linux,
// the function always returns an error on other platforms.
func OriginalTargetAddr(conn net.Conn) (net.Addr, error) {
	return originalTargetAddr(conn)
}
