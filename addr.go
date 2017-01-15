package netx

import (
	"net"
	"strconv"
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

// SplitNetAddr splits the network scheme and the address in s.
func SplitNetAddr(s string) (net string, addr string) {
	if i := strings.Index(s, "://"); i >= 0 {
		net, addr = s[:i], s[i+3:]
	} else {
		addr = s
	}
	return
}

// SplitAddrPort splits the address and port from s.
//
// The function is a wrapper around the standard net.SplitHostPort which
// expects the port part to be a number, setting the port value to -1 if it
// could not parse it.
func SplitAddrPort(s string) (addr string, port int) {
	h, p, err := net.SplitHostPort(s)

	if err != nil {
		addr = s
		port = -1
		return
	}

	if port, err = strconv.Atoi(p); err != nil {
		port = -1
	}

	addr = h
	return
}
