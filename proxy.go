package netx

import (
	"context"
	"fmt"
	"log"
	"net"
)

// Proxyhandler is an interface that must be implemented by types that intend to
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

// A Proxy is a connection handler for intercepted connections.
//
// A proper usage of this proxy type will some iptables rules to redirect TCP
// connections to to the listener this proxy has been setup with.
type Proxy struct {
	// Handler is called by the proxy when it receives a connection that can be
	// proxied.
	//
	// Calling ServeConn on the proxy will panic if this field is nil.
	Handler ProxyHandler

	// ErrorLog is used to log errors detected by the proxy.
	ErrorLog *log.Logger
}

// ServeConn satisfies the Handler interface.
func (p *Proxy) ServeConn(ctx context.Context, conn net.Conn) {
	target, err := OriginalTargetAddr(conn)

	if err != nil {
		p.logf("proxy: %s->%s: %s", conn.RemoteAddr(), conn.LocalAddr(), err)
		return
	}

	p.Handler.ServeProxy(ctx, conn, target)
}

func (p *Proxy) logf(fmt string, args ...interface{}) {
	if log := p.ErrorLog; log != nil {
		log.Printf(fmt, args...)
	}
}

// OriginalTargetAddr returns the original address that an intercepted connection
// intended to reach.
//
// Note that this feature is only available for TCP connections on linux systems.
//
// The function panics if conn is nil.
func OriginalTargetAddr(conn net.Conn) (net.Addr, error) {
	socket, ok := conn.(interface {
		Fd() uintptr
	})

	if !ok {
		return nil, fmt.Errorf("%T has no Fd method, the original destination target address cannot be retrieved", conn)
	}

	return originalTargetAddr(socket.Fd())
}
