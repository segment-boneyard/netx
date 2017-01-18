package netx

import (
	"net"
	"os"
	"syscall"
)

// DupUnix makes a duplicate of the given unix connection.
func DupUnix(conn *net.UnixConn) (*net.UnixConn, error) {
	c, err := dup(conn)
	if err != nil {
		return nil, err
	}
	return c.(*net.UnixConn), nil
}

// DupTCP makes a duplicate of the given TCP connection.
func DupTCP(conn *net.TCPConn) (*net.TCPConn, error) {
	c, err := dup(conn)
	if err != nil {
		return nil, err
	}
	return c.(*net.TCPConn), nil
}

func dup(conn fileConn) (net.Conn, error) {
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	syscall.SetNonblock(int(f.Fd()), true)
	defer f.Close()
	return net.FileConn(f)
}

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

// BasePacketConn returns the base connection object of conn.
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
