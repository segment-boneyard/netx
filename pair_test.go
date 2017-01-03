package netx

import (
	"net"
	"testing"

	"golang.org/x/net/nettest"
)

func TestPair(t *testing.T) {
	for _, network := range [...]string{
		"unix",
		"tcp",
		"tcp4",
		"tcp6",
	} {
		network := network // capture in lambda
		t.Run(network, func(t *testing.T) {
			t.Parallel()
			nettest.TestConn(t, func() (c1 net.Conn, c2 net.Conn, stop func(), err error) {
				if c1, c2, err = Pair(network); err == nil {
					stop = func() {
						c1.Close()
						c2.Close()
					}
				}
				return
			})
		})
	}
}
