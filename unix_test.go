// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package netx

import (
	"net"
	"testing"

	"golang.org/x/net/nettest"
)

func TestSendRecvUnixConn(t *testing.T) {
	nettest.TestConn(t, func() (c3 net.Conn, c4 net.Conn, stop func(), err error) {
		var c1 net.Conn
		var c2 net.Conn
		var u1 *net.UnixConn
		var u2 *net.UnixConn

		if u1, u2, err = UnixConnPair(); err != nil {
			return
		}
		defer u1.Close()
		defer u2.Close()

		if c1, c2, err = ConnPair("tcp"); err != nil {
			return
		}

		if err = SendUnixConn(u1, c1); err != nil {
			return
		}

		if err = SendUnixConn(u2, c2); err != nil {
			return
		}

		if c3, err = RecvUnixConn(u2); err != nil {
			return
		}

		if c4, err = RecvUnixConn(u1); err != nil {
			return
		}

		stop = func() {
			c3.Close()
			c4.Close()
		}
		return
	})
}
