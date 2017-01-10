package netx

import (
	"io"
	"net"
	"testing"
)

func TestTunnel(t *testing.T) {
	net1, addr1, close1 := listenAndServe(Echo)
	defer close1()

	net2, addr2, close2 := listenAndServe(&Proxy{
		Addr: &NetAddr{net1, addr1},
		Handler: &Tunnel{
			Handler: &Forwarder{},
		},
	})
	defer close2()

	conn, err := net.Dial(net2, addr2)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	if _, err := io.WriteString(conn, "Hello World!"); err != nil {
		t.Error(err)
		return
	}

	b := [12]byte{}

	if _, err := io.ReadFull(conn, b[:]); err != nil {
		t.Error(err)
		return
	}

	if s := string(b[:]); s != "Hello World!" {
		t.Error(s)
		return
	}
}
