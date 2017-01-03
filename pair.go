package netx

import (
	"errors"
	"net"
	"os"
	"syscall"
)

// Pair returns a pair of connections, each of them being the end of a
// bidirectional communcation channel. network should be one of "tcp", "tcp4",
// "tcp6", or "unix".
func Pair(network string) (net.Conn, net.Conn, error) {
	switch network {
	case "unix":
		return UnixPair()
	case "tcp", "tcp4", "tcp6":
		return TCPPair(network)
	default:
		return nil, nil, errors.New("unsupported network pair: " + network)
	}
}

// TCPPair returns a pair of TCP connections, each of them being the end of a
// bidirectional communication channel.
func TCPPair(network string) (nc1 *net.TCPConn, nc2 *net.TCPConn, err error) {
	var lstn *net.TCPListener
	var ch1 = make(chan error, 1)
	var ch2 = make(chan *net.TCPConn, 1)

	if lstn, err = net.ListenTCP(network, nil); err != nil {
		return
	}
	defer lstn.Close()

	go func() {
		var conn *net.TCPConn
		var err error

		if conn, err = net.DialTCP(network, nil, lstn.Addr().(*net.TCPAddr)); err != nil {
			ch1 <- err
		} else {
			ch2 <- conn
		}
	}()

	if nc1, err = lstn.AcceptTCP(); err != nil {
		return
	}

	select {
	case nc2 = <-ch2:
	case err = <-ch1:
		nc1.Close()
		nc1 = nil
	}
	return
}

// UnixPair returns a pair of unix connections, each of them being the end of a
// bidirection communication channel..
func UnixPair() (uc1 *net.UnixConn, uc2 *net.UnixConn, err error) {
	var fd1 int
	var fd2 int

	if fd1, fd2, err = socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0); err != nil {
		return
	}

	f1 := os.NewFile(uintptr(fd1), "")
	f2 := os.NewFile(uintptr(fd2), "")

	defer f1.Close()
	defer f2.Close()

	var c1 net.Conn
	var c2 net.Conn

	if c1, err = net.FileConn(f1); err != nil {
		return
	}

	if c2, err = net.FileConn(f2); err != nil {
		c1.Close()
		return
	}

	uc1 = c1.(*net.UnixConn)
	uc2 = c2.(*net.UnixConn)
	return
}
