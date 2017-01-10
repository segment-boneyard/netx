package netx

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

// ProxyHandler is an interface that must be implemented by types that intend to
// proxy connections.
//
// The ServeProxy method is called by a Proxy when it receives a new connection.
// It is similar to the ServeConn method of the Handler interface but receives
// an extra target argument representing the original address that the
// intercepted connection intended to reach.
type ProxyHandler interface {
	ServeProxy(ctx context.Context, conn net.Conn, target net.Addr)
}

// ProxyHandlerFunc makes it possible for simple function types to be used as
// connection proxies.
type ProxyHandlerFunc func(context.Context, net.Conn, net.Addr)

// ServeProxy calls f.
func (f ProxyHandlerFunc) ServeProxy(ctx context.Context, conn net.Conn, target net.Addr) {
	f(ctx, conn, target)
}

// A Proxy is a connection handler that forwards its connections to a proxy
// handler.
type Proxy struct {
	// Network and Address represent the target to which the proxy is forwarding
	// connections.
	Network string
	Address string

	// Handler is the proxy handler to which connetions are forwarded to.
	Handler ProxyHandler
}

// ServeConn satsifies the Handler interface.
func (p *Proxy) ServeConn(ctx context.Context, conn net.Conn) {
	p.Handler.ServeProxy(ctx, conn, &NetAddr{
		Net:  p.Network,
		Addr: p.Address,
	})
}

// A TransparentProxy is a connection handler for intercepted connections.
//
// A proper usage of this proxy requires some iptables rules to redirect TCP
// connections to to the listener its attached to.
type TransparentProxy struct {
	// Handler is called by the proxy when it receives a connection that can be
	// proxied.
	//
	// Calling ServeConn on the proxy will panic if this field is nil.
	Handler ProxyHandler
}

// ServeConn satisfies the Handler interface.
//
// The method panics to report errors.
func (p *TransparentProxy) ServeConn(ctx context.Context, conn net.Conn) {
	target, err := OriginalTargetAddr(conn)
	if err != nil {
		panic(err)
	}
	p.Handler.ServeProxy(ctx, conn, target)
}

// OriginalTargetAddr returns the original address that an intercepted
// connection intended to reach.
//
// Note that this feature is only available for TCP connections on linux,
// the function always returns an error on other platforms.
func OriginalTargetAddr(conn net.Conn) (net.Addr, error) {
	return originalTargetAddr(conn)
}

// ProxyProtoHandler is the implementation of a connection handler which speaks
// the proxy protocol.
//
// When the handler receives a LOCAL connection it attempts to protomote its
// handler to a Handler to serve it the connection. If this doesn't work the
// connection is simply closed.
//
// Version 1 and 2 are supported.
//
// http://www.haproxy.org/download/1.5/doc/proxy-protocol.txt
type ProxyProtoHandler struct {
	Handler ProxyHandler
}

// ServeConn satisifies the Handler interface.
func (p *ProxyProtoHandler) ServeConn(ctx context.Context, conn net.Conn) {
	src, dst, buf, local, err := parseProxyProto(conn)

	if err != nil {
		panic(err)
	}

	if local {
		if h, ok := p.Handler.(Handler); ok {
			h.ServeConn(ctx, conn)
		} else {
			conn.Close()
		}
		return
	}

	proxyConn := &proxyProtoConn{
		Conn: conn,
		src:  src,
		buf:  buf,
	}
	p.Handler.ServeProxy(ctx, proxyConn, dst)
}

type proxyProtoConn struct {
	net.Conn
	src net.Addr
	buf []byte
}

func (c *proxyProtoConn) RemoteAddr() net.Addr {
	return c.src
}

func (c *proxyProtoConn) Read(b []byte) (n int, err error) {
	if len(c.buf) != 0 {
		n = copy(b, c.buf)
		c.buf = c.buf[n:]
		return
	}
	return c.Conn.Read(b)
}

var (
	proxy     = [...]byte{'P', 'R', 'O', 'X', 'Y'}
	tcp4      = [...]byte{'T', 'C', 'P', '4'}
	tcp6      = [...]byte{'T', 'C', 'P', '6'}
	crlf      = [...]byte{'\r', '\n'}
	signature = [...]byte{'\x0D', '\x0A', '\x0D', '\x0A', '\x00', '\x0D', '\x0A', '\x51', '\x55', '\x49', '\x54', '\x0A'}
)

func appendProxyProtoV1(b []byte, src net.Addr, dst net.Addr) []byte {
	var srcPortBuf [8]byte
	var dstPortBuf [8]byte
	var family []byte

	srcTCP := src.(*net.TCPAddr)
	dstTCP := dst.(*net.TCPAddr)

	srcPort := strconv.AppendUint(srcPortBuf[:0], uint64(srcTCP.Port), 10)
	dstPort := strconv.AppendUint(dstPortBuf[:0], uint64(dstTCP.Port), 10)

	if srcTCP.IP.To4() != nil {
		family = tcp4[:]
	} else {
		family = tcp6[:]
	}

	b = append(b, proxy[:]...)
	b = append(b, ' ')
	b = append(b, family...)
	b = append(b, ' ')
	b = append(b, srcTCP.IP.String()...)
	b = append(b, ' ')
	b = append(b, dstTCP.IP.String()...)
	b = append(b, ' ')
	b = append(b, srcPort...)
	b = append(b, ' ')
	b = append(b, dstPort...)
	b = append(b, crlf[:]...)
	return b
}

func appendProxyProtoV2(b []byte, src net.Addr, dst net.Addr, local bool) []byte {
	const (
		AF_UNSPEC = 0
		AF_INET   = 1
		AF_INET6  = 2
		AF_UNIX   = 3

		UNSPEC = 0
		STREAM = 1
		DGRAM  = 2

		PROXY   = 1
		VERSION = 2
	)

	var (
		vercmd   byte = VERSION << 4
		family   byte = AF_UNSPEC
		socktype byte = UNSPEC

		srcIP net.IP
		dstIP net.IP

		srcAddr []byte
		dstAddr []byte
		srcPort []byte
		dstPort []byte

		srcAddrBuf [108]byte
		dstAddrBuf [108]byte
		srcPortBuf [2]byte
		dstPortBuf [2]byte
	)

	if !local {
		vercmd |= PROXY
	}

	switch a := src.(type) {
	case *net.TCPAddr:
		b := dst.(*net.TCPAddr)
		socktype, srcIP, dstIP = STREAM, a.IP, b.IP
		srcPort = srcPortBuf[:]
		dstPort = dstPortBuf[:]
		binary.BigEndian.PutUint16(srcPort, uint16(a.Port))
		binary.BigEndian.PutUint16(dstPort, uint16(b.Port))

	case *net.UDPAddr:
		b := dst.(*net.UDPAddr)
		socktype, srcIP, dstIP = DGRAM, a.IP, b.IP
		srcPort = srcPortBuf[:]
		dstPort = dstPortBuf[:]
		binary.BigEndian.PutUint16(srcPort, uint16(a.Port))
		binary.BigEndian.PutUint16(dstPort, uint16(b.Port))

	case *net.UnixAddr:
		b := dst.(*net.UnixAddr)
		family = AF_UNIX
		srcAddr = srcAddrBuf[:]
		dstAddr = dstAddrBuf[:]
		copy(srcAddr, a.Name)
		copy(dstAddr, b.Name)

		switch a.Net {
		case "unix":
			socktype = STREAM
		case "unixgram":
			socktype = DGRAM
		}
	}

	if srcIP != nil {
		if ip := srcIP.To4(); ip != nil {
			family = AF_INET
			srcAddr = ip
			dstAddr = dstIP.To4()
		} else {
			family = AF_INET6
			srcAddr = srcIP.To16()
			dstAddr = dstIP.To16()
		}
	}

	b = append(b, signature[:]...)
	b = append(b, vercmd)
	b = append(b, (family<<4)|socktype)
	b = append(b, srcAddr...)
	b = append(b, dstAddr...)
	b = append(b, srcPort...)
	b = append(b, dstPort...)
	return b
}

func parseProxyProto(r io.Reader) (src net.Addr, dst net.Addr, buf []byte, local bool, err error) {
	var a [256]byte
	var b []byte
	var n int

	if n, err = io.ReadAtLeast(r, a[:], 14); err != nil {
		return
	}
	b = a[:n]

	switch {
	case bytes.HasPrefix(b, proxy[:]):
		var i = bytes.Index(b[:107], crlf[:])

		for i < 0 {
			if len(b) >= 107 {
				err = errors.New("no '\r\n' sequence found in the first 107 bytes of a proxy protocol connection")
				return
			}
			if n, err = r.Read(a[len(b):107]); n == 0 {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return
			}
			b = a[:len(b)+n]
			i = bytes.Index(b, crlf[:])
		}

		src, dst, err = parseProxyProtoV1(b[:i])
		buf = b[i+2:]
		return

	case bytes.HasPrefix(b, signature[:]):
		b = b[len(signature):]

		if version := b[0] >> 4; version != 2 {
			err = fmt.Errorf("invalid proxy protocol version: %#d", version)
			return
		}

		switch cmd := b[0] & 0xF; cmd {
		case 0:
			local = true
		case 1:
		default:
			err = fmt.Errorf("invalid proxy protocol command: %#x", cmd)
			return
		}

		var makeStreamAddr = makeTCPAddr
		var makeDgramAddr = makeUDPAddr
		var makeAddr func(int, []byte, []byte) net.Addr
		var addrLen int
		var portLen int
		var socktype int

		switch family := b[1] >> 4; family {
		case 0: // AF_UNSPEC
		case 1: // AF_INET
			addrLen, portLen = 4, 2
		case 2: // AF_INET6
			addrLen, portLen = 16, 2
		case 3: // AF_UNIX
			addrLen, portLen = 108, 0
			makeStreamAddr, makeDgramAddr = makeUnixAddr, makeUnixAddr
		default:
			err = fmt.Errorf("invalid socket family found in proxy protocol header: %#x", family)
			return
		}

		switch socktype = int(b[1] & 0xF); socktype {
		case 0: // UNSPEC
		case 1: // STREAM
			makeAddr = makeStreamAddr
		case 2: // DGRAM
			makeAddr = makeDgramAddr
		default:
			err = fmt.Errorf("invalid socket type found in proxy protocol header: %#x", socktype)
			return
		}
		b = b[2:]

		n1 := 2*addrLen + 2*portLen
		n2 := len(b)

		if n1 > n2 {
			if _, err = io.ReadFull(r, b[n2:n1]); err != nil {
				return
			}
			b = b[:n1]
		}

		if makeAddr != nil {
			src = makeAddr(socktype, b[:addrLen], b[2*addrLen:2*addrLen+portLen])
			dst = makeAddr(socktype, b[addrLen:2*addrLen], b[2*addrLen+portLen:])
		}

		buf = b[n1:]
		return
	}

	err = errors.New("invalid signature found in proxy protocol connection")
	return
}

func parseProxyProtoV1(b []byte) (src net.Addr, dst net.Addr, err error) {
	var family, srcIP, srcPort, dstIP, dstPort []byte

	if !bytes.HasPrefix(b, proxy[:]) {
		err = errors.New("expected 'PROXY' at the beginning of the proxy protocol connection")
		return
	}

	if b = b[len(proxy):]; len(b) != 0 && b[0] == ' ' {
		b = b[1:]
	}

	family, b = parseProxyProtoWord(b)
	srcIP, b = parseProxyProtoWord(b)
	dstIP, b = parseProxyProtoWord(b)
	srcPort, b = parseProxyProtoWord(b)
	dstPort, b = parseProxyProtoWord(b)

	switch {
	case bytes.Equal(family, tcp4[:]):
	case bytes.Equal(family, tcp6[:]):
	default:
		err = fmt.Errorf("invalid socket family found in proxy protocol header: %s", string(family))
		return
	}

	var srcTCP net.TCPAddr
	var dstTCP net.TCPAddr

	if srcTCP.IP = net.ParseIP(string(srcIP)); srcTCP.IP == nil {
		err = fmt.Errorf("invalid source address found in proxy protocol header: %s", string(srcIP))
		return
	}

	if dstTCP.IP = net.ParseIP(string(dstIP)); dstTCP.IP == nil {
		err = fmt.Errorf("invalid destination address found in proxy protocol header: %s", string(dstIP))
		return
	}

	if srcTCP.Port, err = strconv.Atoi(string(srcPort)); err != nil {
		err = fmt.Errorf("invalid source port found in proxy protocol header: %s", string(srcPort))
		return
	}

	if dstTCP.Port, err = strconv.Atoi(string(dstPort)); err != nil {
		err = fmt.Errorf("invalid source port found in proxy protocol header: %s", string(dstPort))
		return
	}

	if len(b) != 0 {
		err = errors.New("invalid extra bytes found at the end of a proxy protocol header")
		return
	}

	src, dst = &srcTCP, &dstTCP
	return
}

func parseProxyProtoWord(b []byte) (word []byte, tail []byte) {
	for i, n := 0, len(b); i != n; i++ {
		if b[i] == ' ' {
			word, tail = b[:i], b[i+1:]
			return
		}
	}
	word = b
	return
}

func makeTCPAddr(_ int, ip []byte, port []byte) net.Addr {
	return &net.TCPAddr{
		IP:   makeIP(ip),
		Port: int(binary.BigEndian.Uint16(port)),
	}
}

func makeUDPAddr(_ int, ip []byte, port []byte) net.Addr {
	return &net.UDPAddr{
		IP:   makeIP(ip),
		Port: int(binary.BigEndian.Uint16(port)),
	}
}

func makeUnixAddr(socktype int, name []byte, _ []byte) net.Addr {
	off := bytes.IndexByte(name, 0)
	if off < 0 {
		off = len(name)
	}

	addr := &net.UnixAddr{
		Name: string(name[:off]),
	}

	switch {
	case socktype == 1: // STREAM
		addr.Net = "unix"
	case socktype == 2: // DGRAM
		addr.Net = "unixgram"
	}

	return addr
}

func makeIP(b []byte) net.IP {
	if len(b) == 4 {
		return net.IPv4(b[0], b[1], b[2], b[3])
	}
	ip := make(net.IP, len(b))
	copy(ip, b)
	return ip
}
