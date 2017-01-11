package netx

import (
	"context"
	"net"
	"sync"
)

// Forwarder is a tunnel handler that simply passes bytes between the two ends
// of a tunnel.
type Forwarder struct{}

// ServeTunnel satisfies the TunnelHandler interface.
func (t *Forwarder) ServeTunnel(ctx context.Context, from net.Conn, to net.Conn) {
	join := &sync.WaitGroup{}
	join.Add(2)

	go t.forward(from, to, join)
	go t.forward(to, from, join)

	<-ctx.Done()
}

func (t *Forwarder) forward(from net.Conn, to net.Conn, join *sync.WaitGroup) {
	defer join.Done()
	Copy(to, from)
}
