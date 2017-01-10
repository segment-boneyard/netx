package netx

import (
	"context"
	"net"
	"time"
)

// TunnelHandler is an interface that must be implemented by types that intend
// to provide logic for tunnelling connections.
//
// The ServeTunnel method is called by a Tunnel after establishing a connection
// to a remote target address.
type TunnelHandler interface {
	ServeTunnel(ctx context.Context, from net.Conn, to net.Conn)
}

// TunnelHandlerFunc makes it possible for simple function types to be used as
// connection proxies.
type TunnelHandlerFunc func(context.Context, net.Conn, net.Conn)

// ServeTunnel calls f.
func (f TunnelHandlerFunc) ServeTunnel(ctx context.Context, from net.Conn, to net.Conn) {
	f(ctx, from, to)
}

// A Tunnel is a proxy handler that establishes a second connection to a
// target address for every incoming connection it receives.
type Tunnel struct {
	// Handler is called by the tunnel when it successfully established a
	// connection to its target.
	//
	// Calling one of the tunnel's method will panic if this field is nil.
	Handler TunnelHandler

	// DialContext can be set to a dialing function to configure how the tunnel
	// establishes new connections.
	DialContext func(context.Context, string, string) (net.Conn, error)
}

// ServeProxy satisfies the ProxyHandler interface.
//
// When called the tunnel establishes a connection to target, then delegate to
// its handler.
//
// The method panics to report errors.
func (t *Tunnel) ServeProxy(ctx context.Context, from net.Conn, target net.Addr) {
	dial := t.DialContext

	if dial == nil {
		dial = (&net.Dialer{Timeout: 1 * time.Minute /* safeguard */}).DialContext
	}

	to, err := dial(ctx, target.Network(), target.String())

	if err != nil {
		panic(err)
	}

	t.Handler.ServeTunnel(ctx, from, to)
}
