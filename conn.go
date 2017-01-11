package netx

import (
	"io"
	"net"
	"os"
)

// BaseConn returns the base connection object of conn.
//
// The function works by dynamically checking whether conn implements the
// `BaseConn() net.Conn` method, recursing dynamically to find the root connection
// object.
func BaseConn(conn net.Conn) net.Conn {
	type base interface {
		Base() net.Conn
	}

	for ok := true; ok; {
		var b base
		if b, ok = conn.(base); ok {
			conn = b.Base()
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
	type base interface {
		BasePacket() net.PacketConn
	}

	for ok := true; ok; {
		var b base
		if b, ok = conn.(base); ok {
			conn = b.BasePacket()
		}
	}

	return conn
}

// fileConn is used internally to figure out if a net.Conn value also exposes a
// File method.
type fileConn interface {
	File() (*os.File, error)
}

// readCloser is an interface implemented by connections that can be closed only
// on their read end.
type readCloser interface {
	CloseRead() error
}

func closeRead(c io.Closer) error {
	if rc, ok := c.(readCloser); ok {
		return rc.CloseRead()
	}
	return c.Close()
}
