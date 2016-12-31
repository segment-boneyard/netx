package netx

import (
	"net"
	"strings"
)

// NetAddr is a type satisifying the net.Addr interface.
type NetAddr struct {
	Net  string
	Addr string
}

// Network returns a.Net
func (a *NetAddr) Network() string { return a.Net }

// String returns a.Addr
func (a *NetAddr) String() string { return a.Addr }

// MultiAddr is used for compound listeners returned by MultiListener.
type MultiAddr []net.Addr

// Network returns a comma-separated list of the addresses networks.
func (addr MultiAddr) Network() string {
	s := make([]string, len(addr))
	for i, a := range addr {
		s[i] = a.Network()
	}
	return strings.Join(s, ",")
}

// String returns a comma-separated list of the addresses string
// representations.
func (addr MultiAddr) String() string {
	s := make([]string, len(addr))
	for i, a := range addr {
		s[i] = a.String()
	}
	return strings.Join(s, ",")
}
