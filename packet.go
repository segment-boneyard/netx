package netx

import (
	"context"
	"net"
)

// A PacketHandler handles packets received from packet connections.
type PacketHandler interface {
	ServePacket(conn net.PacketConn, from net.Addr, bytes []byte, context context.Context)
}
