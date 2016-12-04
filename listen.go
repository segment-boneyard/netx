package netx

import (
	"errors"
	"net"
	"strings"
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
	var host string
	var port string
	var ifi *net.Interface
	var addrs = make([]string, 0, 10)

	if off := strings.Index(address, "://"); off >= 0 {
		for _, proto := range [...]string{
			"tcp4",
			"tcp6",
			"tcp",
			"unixpacket",
			"unix",
		} {
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
			network = "tcp"
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
			network = "tcp"
		}

	} else {
		// Neither an IP address nor a network interface name was passed, we
		// assume this address is probably the path to a unix domain socket.
		addrs = append(addrs, address)

		if len(network) == 0 {
			network = "unix"
		}
	}

	// TOOD: listen on all addresses?
	for _, address := range addrs {
		if lstn, err = net.Listen(network, address); err == nil {
			break
		}
	}

	return
}
