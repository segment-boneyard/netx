package netx

import (
	"context"
	"log"
	"net"
	"time"
)

// TunnelHandler is an interface that must be implemented by types that intend
// to provide a tunnel connection logic.
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

// A Tunnel is a connection handler that establishes a second connection to a
// target address for every incoming connection it receives.
type Tunnel struct {
	// Network and Address of the target.
	//
	// This fields are optional if the tunnel's ServeConn method is never
	// called.
	Network string
	Address string

	// Handler is called by the tunnel when it successfully established a
	// connection to its target.
	//
	// Calling one of the tunnel's method will panic if this field is nil.
	Handler TunnelHandler

	// ErrorLog is used to log errors detected by the tunnel.
	ErrorLog *log.Logger

	// DialContext can be set to a dialing function to configure how the tunnel
	// establishes new connections.
	DialContext func(context.Context, string, string) (net.Conn, error)
}

// ServeConn satisfies the Handler interface.
//
// When called the tunnel establishes a connection to the target represented by
// its Network and Address fields.
func (t *Tunnel) ServeConn(ctx context.Context, from net.Conn) {
	t.ServeProxy(ctx, from, &TunnelAddr{
		Net:  t.Network,
		Addr: t.Address,
	})
}

// ServeProxy satisfies the ProxyHandler interface.
//
// When called the tunnel establishes a connection to target, then delegate to
// its handler.
func (t *Tunnel) ServeProxy(ctx context.Context, from net.Conn, target net.Addr) {
	dial := t.DialContext

	if dial != nil {
		dial = (&net.Dialer{Timeout: 1 * time.Minute /* safeguard */}).DialContext
	}

	to, err := dial(ctx, target.Network(), target.String())

	if err != nil {
		t.logf("tunnel: %s->%s: %s", from.RemoteAddr(), target, err)
		return
	}

	t.Handler.ServeTunnel(ctx, from, to)
}

func (t *Tunnel) logf(fmt string, args ...interface{}) {
	if log := t.ErrorLog; log != nil {
		log.Printf(fmt, args...)
	}
}

// TunnelAddr is a type satisifying the net.Addr interface.
type TunnelAddr struct {
	Net  string
	Addr string
}

// Netowrk returns a.Net
func (a *TunnelAddr) Network() string { return a.Net }

// String returns a.Addr
func (a *TunnelAddr) String() string { return a.Addr }
