package netx

import "net"

// A Handler manages a network connection.
type Handler interface {
	ServeConn(conn net.Conn)
}
