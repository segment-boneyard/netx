package netx

import (
	"io"
	"net"
	"testing"

	"golang.org/x/net/nettest"
)

func TestConnPair(t *testing.T) {
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
				if c1, c2, err = ConnPair(network); err == nil {
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

func TestUnixConnPairCloseWrite(t *testing.T) {
	c1, c2, err := UnixConnPair()
	if err != nil {
		t.Error(err)
		return
	}
	defer c1.Close()
	defer c2.Close()

	b := make([]byte, 100)

	if err := c1.CloseWrite(); err != nil {
		t.Error(err)
		return
	}

	if _, err := c2.Read(b); err != io.EOF {
		t.Error("expected EOF but got", err)
		return
	}

	if _, err := c2.Write([]byte("Hello World!")); err != nil {
		t.Error(err)
		return
	}

	if n, err := c1.Read(b); err != nil {
		t.Error(err)
		return
	} else if n != 12 {
		t.Error("bad number of byts returned:", n)
		return
	}

	if s := string(b[:12]); s != "Hello World!" {
		t.Error("bad content read:", s)
		return
	}
}
