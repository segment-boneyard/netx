package netx

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
)

// Forwarder is a tunnel handler that simply passes bytes between the two ends
// of a tunnel.
type Forwarder struct{}

// CanRead satisfies the ProtoReader interface, always returns true. This means
// that a forwarder can be used as a fallback protocol in a TunnelProtoMux to
// simply pass the bytes back and forth.
func (t *Forwarder) CanRead(b []byte) bool {
	return true
}

// ServeTunnel satisfies the TunnelHandler interface.
func (t *Forwarder) ServeTunnel(ctx context.Context, from net.Conn, to net.Conn) {
	defer from.Close()
	defer to.Close()

	join := &sync.WaitGroup{}
	join.Add(2)

	go t.forward(from, to, join)
	go t.forward(to, from, join)

	<-ctx.Done()
}

func (t *Forwarder) forward(from net.Conn, to net.Conn, join *sync.WaitGroup) {
	defer join.Done()
	defer to.Close()

	buf := buffers.Get().(*bytes.Buffer)
	io.CopyBuffer(to, from, buf.Bytes())
	buffers.Put(buf)
}

var buffers = sync.Pool{
	New: func() interface{} { return bytes.NewBuffer(make([]byte, 16384, 16384)) },
}
