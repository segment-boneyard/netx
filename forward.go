package netx

import (
	"context"
	"net"
)

// Forwarder is a tunnel handler that simply passes bytes between the two ends
// of a tunnel.
type Forwarder struct{}

// ServeTunnel satisfies the TunnelHandler interface.
func (t *Forwarder) ServeTunnel(ctx context.Context, from net.Conn, to net.Conn) {
	defer to.Close()
	defer from.Close()

	go Copy(to, from)
	go Copy(from, to)

	<-ctx.Done()
}
