package netx

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"
)

// Listen is equivalent to net.Listen but guesses the network from the address.
//
// The function accepts addresses that may be prefixed by a URL scheme to set
// the protocol that will be used, supported protocols are tcp, tcp4, tcp6,
// unix, and unixpacket.
//
// The address may contain a path to a file for unix sockets, a pair of an IP
// address and port, a pair of a network interface name and port, or just port.
//
// If the port is omitted for network addresses the operating system will pick
// one automatically.
func Listen(address string) (lstn net.Listener, err error) {
	var network string
	var addrs []string

	if network, addrs, err = resolveListen(address, "tcp", "unix", []string{
		"tcp",
		"tcp4",
		"tcp6",
		"unix",
		"unixpacket",
	}); err != nil {
		return
	}

	if len(addrs) == 1 {
		return net.Listen(network, addrs[0])
	}

	lstns := make([]net.Listener, 0, len(addrs))

	for _, a := range addrs {
		l, e := net.Listen(network, a)
		if e != nil {
			for _, l := range lstns {
				l.Close()
			}
			return
		}
		lstns = append(lstns, l)
	}

	lstn = MultiListener(lstns...)
	return
}

// ListenPacket is similar to Listen but returns a PacketConn, nad works with
// udp, udp4, udp6, ip, ip4, ip6, or unixdgram protocols.
func ListenPacket(address string) (conn net.PacketConn, err error) {
	var network string
	var addrs []string

	if network, addrs, err = resolveListen(address, "udp", "unixdgram", []string{
		"udp",
		"udp4",
		"udp6",
		"ip",
		"ip4",
		"ip6",
		"unixdgram",
	}); err != nil {
		return
	}

	// TODO: listen on all addresses?
	for _, a := range addrs {
		if conn, err = net.ListenPacket(network, a); err == nil {
			break
		}
	}

	return
}

func resolveListen(address string, defaultProtoNetwork string, defaultProtoUnix string, protocols []string) (network string, addrs []string, err error) {
	var host string
	var port string
	var ifi *net.Interface

	if off := strings.Index(address, "://"); off >= 0 {
		for _, proto := range protocols {
			if strings.HasPrefix(address, proto+"://") {
				network, address = proto, address[len(proto)+3:]
				break
			}
		}

		if len(network) == 0 {
			err = errors.New("unsupported protocol: " + address[:off])
			return
		}
	}

	if host, port, err = net.SplitHostPort(address); err != nil {
		err = nil

		if strings.HasPrefix(address, ":") {
			// the address doesn't mention which interface to listen on
			port = address[1:]
		} else {
			// the address doesn't mention which port to listen on
			host = address
		}
	}

	if IsIP(host) {
		// The function received a simple IP address to listen on.
		addrs = append(addrs, address)

		if len(network) == 0 {
			network = defaultProtoNetwork
		}

	} else if ifi, err = net.InterfaceByName(host); err == nil {
		// The function received the name of a network interface, we have to
		// lookup the list of all network addresses to listen on.
		var ifa []net.Addr

		if ifa, err = ifi.Addrs(); err != nil {
			return
		}

		for _, a := range ifa {
			s := a.String()
			if len(port) != 0 {
				s = net.JoinHostPort(s, port)
			}
			addrs = append(addrs, s)
		}

		if len(network) == 0 {
			network = defaultProtoNetwork
		}

	} else {
		// Neither an IP address nor a network interface name was passed, we
		// assume this address is probably the path to a unix domain socket.
		addrs = append(addrs, address)

		if len(network) == 0 {
			network = defaultProtoUnix
		}
	}

	return
}

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

// MultiListener returns a compound listener made of the given list of
// listeners.
func MultiListener(lstn ...net.Listener) net.Listener {
	c := make(chan net.Conn)
	e := make(chan error)
	d := make(chan struct{})
	x := make(chan struct{})
	m := &multiListener{
		l: append(make([]net.Listener, 0, len(lstn)), lstn...),
		c: c,
		e: e,
		d: d,
		x: x,
	}

	for _, l := range m.l {
		go func(l net.Listener, c chan<- net.Conn, e chan<- error, d chan<- struct{}) {
			defer func() { d <- struct{}{} }()
			for {
				if conn, err := l.Accept(); err == nil {
					c <- conn
				} else {
					e <- err

					if !IsTemporary(err) {
						break
					}
				}
			}
		}(l, c, e, d)
	}

	return m
}

type multiListener struct {
	l []net.Listener  // the list of listeners
	c <-chan net.Conn // connections from Accept are published on this channel
	e <-chan error    // errors from Accept are published on this channel
	d <-chan struct{} // each goroutine publishes to this channel when they exit
	x chan struct{}   // closed when the listener is closed

	// Used by Close to allow multiple goroutines to call the method as well as
	// allowing the method to be called multiple times.
	once sync.Once
}

func (m *multiListener) Accept() (conn net.Conn, err error) {
	select {
	case conn = <-m.c:
	case err = <-m.e:
	case <-m.x:
		err = io.ErrClosedPipe
	}
	return
}

func (m *multiListener) Close() (err error) {
	m.once.Do(func() {
		var errs []string

		for _, l := range m.l {
			if e := l.Close(); e != nil {
				errs = append(errs, e.Error())
			}
		}

		for i, n := 0, len(m.l); i != n; {
			select {
			case conn := <-m.c:
				conn.Close()
			case <-m.e:
			case <-m.d:
				i++
			}
		}

		if errs != nil {
			err = errors.New(strings.Join(errs, "; "))
		}

		close(m.x)
	})
	return
}

func (m *multiListener) Addr() net.Addr {
	a := make(MultiAddr, len(m.l))

	for i, l := range m.l {
		a[i] = l.Addr()
	}

	return a
}
