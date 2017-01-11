package netx

import (
	"net"
	"os"
)

// BaseConn returns the base connection object of conn.
//
// The function works by dynamically checking whether conn implements the
// `BaseConn() net.Conn` method, recursing dynamically to find the root connection
// object.
func BaseConn(conn net.Conn) net.Conn {
	for ok := true; ok; {
		var b baseConn
		if b, ok = conn.(baseConn); ok {
			conn = b.BaseConn()
		}
	}
	return conn
}

// BaseConn returns the base connection object of conn.
//
// The function works by dynamically checking whether conn implements the
// `BasePacketConn() net.PacketConn` method, recursing dynamically to find the root connection
// object.
func BasePacketConn(conn net.PacketConn) net.PacketConn {
	for ok := true; ok; {
		var b basePacketConn
		if b, ok = conn.(basePacketConn); ok {
			conn = b.BasePacketConn()
		}
	}
	return conn
}

// baseConn is an interface implemented by connection wrappers wanting to expose
// the underlying net.Conn object they use.
type baseConn interface {
	BaseConn() net.Conn
}

// basePacketConn is an interface implemented by connection wrappers wanting to
// expose the underlying net.PacketConn object they use.
type basePacketConn interface {
	BasePacketConn() net.PacketConn
}

// fileConn is used internally to figure out if a net.Conn value also exposes a
// File method.
type fileConn interface {
	File() (*os.File, error)
}
