package netx

import (
	"net"
	"testing"
)

type baseTestConn struct{ net.Conn }

func (c *baseTestConn) BaseConn() net.Conn { return c.Conn }

func TestBaseConn(t *testing.T) {
	c1 := &net.TCPConn{}
	c2 := &baseTestConn{c1}

	tests := []struct {
		name string
		base net.Conn
		conn net.Conn
	}{
		{
			name: "base:false",
			base: c1,
			conn: c1,
		},
		{
			name: "base:true",
			base: c1,
			conn: c2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if base := BaseConn(test.conn); base != test.base {
				t.Errorf("bad base conn: %#v", base)
			}
		})
	}
}

type baseTestPacketConn struct{ net.PacketConn }

func (c *baseTestPacketConn) BasePacketConn() net.PacketConn { return c.PacketConn }

func TestBasePacketConn(t *testing.T) {
	c1 := &net.UDPConn{}
	c2 := &baseTestPacketConn{c1}

	tests := []struct {
		name string
		base net.PacketConn
		conn net.PacketConn
	}{
		{
			name: "base:false",
			base: c1,
			conn: c1,
		},
		{
			name: "base:true",
			base: c1,
			conn: c2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if base := BasePacketConn(test.conn); base != test.base {
				t.Errorf("bad base packet conn: %#v", base)
			}
		})
	}
}
