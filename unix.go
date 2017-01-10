package netx

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
	"syscall"
)

// fileConn is used internally to figure out if a net.Conn value also exposes a
// File method.
type fileConn interface {
	File() (*os.File, error)
}

// SendUnixConn sends a file descriptor embedded in conn over the unix domain
// socket.
// On success conn is closed because the owner is now the process that received
// the file descriptor.
//
// conn must be a *net.TCPConn or similar (providing a File method) or the
// function will panic.
func SendUnixConn(socket *net.UnixConn, conn net.Conn) (err error) {
	return sendUnixFileConn(socket, conn.(fileConn), conn)
}

// SendUnixPacketConn sends a file descriptor embedded in conn over the unix
// domain socket.
// On success conn is closed because the owner is now the process that received
// the file descriptor.
//
// conn must be a *net.UDPConn or similar (providing a File method) or the
// function will panic.
func SendUnixPacketConn(socket *net.UnixConn, conn net.PacketConn) (err error) {
	return sendUnixFileConn(socket, conn.(fileConn), conn)
}

func sendUnixFileConn(socket *net.UnixConn, conn fileConn, close io.Closer) (err error) {
	var f *os.File

	if f, err = conn.File(); err != nil {
		return
	}
	defer f.Close()

	if err = SendUnixFile(socket, f); err != nil {
		return
	}

	close.Close()
	return
}

// SendUnixFile sends a file descriptor embedded in file over the unix domain
// socket.
// On success the file is closed because the owner is now the process that
// received the file descriptor.
func SendUnixFile(socket *net.UnixConn, file *os.File) (err error) {
	var fds = [1]int{int(file.Fd())}
	var oob = syscall.UnixRights(fds[:]...)

	if _, _, err = socket.WriteMsgUnix(nil, oob, nil); err != nil {
		return
	}

	file.Close()
	return
}

// RecvUnixConn receives a network connection from a unix domain socket.
func RecvUnixConn(socket *net.UnixConn) (conn net.Conn, err error) {
	var f *os.File
	if f, err = RecvUnixFile(socket); err != nil {
		return
	}
	defer f.Close()
	return net.FileConn(f)
}

// RecvUnixPacketConn receives a packet oriented network connection from a unix
// domain socket.
func RecvUnixPacketConn(socket *net.UnixConn) (conn net.PacketConn, err error) {
	var f *os.File
	if f, err = RecvUnixFile(socket); err != nil {
		return
	}
	defer f.Close()
	return net.FilePacketConn(f)
}

// RecvUnixFile receives a file descriptor from a unix domain socket.
func RecvUnixFile(socket *net.UnixConn) (file *os.File, err error) {
	var oob = make([]byte, syscall.CmsgSpace(4))
	var oobn int
	var msg []syscall.SocketControlMessage
	var fds []int

	if _, oobn, _, _, err = socket.ReadMsgUnix(nil, oob); err != nil {
		return
	} else if oobn == 0 {
		err = io.EOF
		return
	}

	if msg, err = syscall.ParseSocketControlMessage(oob); err != nil {
		err = os.NewSyscallError("ParseSocketControlMessage", err)
		return
	}

	if len(msg) != 1 {
		err = fmt.Errorf("invalid number of socket control messages, expected 1 but found %d", len(msg))
		return
	}

	if fds, err = syscall.ParseUnixRights(&msg[0]); err != nil {
		err = os.NewSyscallError("ParseUnixRights", err)
		return
	}

	if len(fds) != 1 {
		for _, fd := range fds {
			syscall.Close(fd)
		}
		err = fmt.Errorf("too many file descriptors found in a single control message, %d were closed", len(fds))
		return
	}

	file = os.NewFile(uintptr(fds[0]), "")
	return
}

// NewRecvUnixListener returns a new listener which accepts connection by
// reading file descriptors from a unix domain socket.
//
// The function doesn't make a copy of socket, so the returned listener should
// be considered the new owner of that object, which means closing the listener
// will actually close the original socket (and vice versa).
func NewRecvUnixListener(socket *net.UnixConn) *RecvUnixListener {
	return &RecvUnixListener{*socket}
}

// RecvUnixListener is a listener which acceptes connections by reading file
// descriptors from a unix domain socket.
type RecvUnixListener struct {
	socket net.UnixConn
}

// Accept receives a file descriptor from the listener's unix domain socket.
func (l *RecvUnixListener) Accept() (net.Conn, error) {
	return RecvUnixConn(&l.socket)
}

// Addr returns the address of the listener's unix domain socket.
func (l *RecvUnixListener) Addr() net.Addr {
	return l.socket.LocalAddr()
}

// Close closes the underlying unix domain socket.
func (l *RecvUnixListener) Close() error {
	return l.socket.Close()
}

// NewSendUnixHandler wraps handler so the connetions it receives will be sent
// back to socket when handler returns without closing them.
func NewSendUnixHandler(socket *net.UnixConn, handler Handler) *SendUnixHandler {
	return &SendUnixHandler{
		handler: handler,
		socket:  *socket,
	}
}

// SendUnixHandler is a connection handler which sends the connections it
// handles back through a unix domain socket.
type SendUnixHandler struct {
	handler Handler
	socket  net.UnixConn
}

// ServeConn satisfies the Handler interface.
func (h *SendUnixHandler) ServeConn(ctx context.Context, conn net.Conn) {
	c := &sendUnixConn{Conn: conn}
	defer func() {
		if atomic.LoadUint32(&c.closed) == 0 {
			SendUnixConn(&h.socket, conn)
		}
	}()
	h.handler.ServeConn(ctx, c)
}

type sendUnixConn struct {
	net.Conn
	closed uint32
}

func (c *sendUnixConn) Close() (err error) {
	atomic.StoreUint32(&c.closed, 1)
	return c.Conn.Close()
}

func (c *sendUnixConn) Read(b []byte) (n int, err error) {
	if n, err = c.Conn.Read(b); err != nil && !IsTemporary(err) {
		atomic.StoreUint32(&c.closed, 1)
	}
	return
}

func (c *sendUnixConn) Write(b []byte) (n int, err error) {
	if n, err = c.Conn.Write(b); err != nil && !IsTemporary(err) {
		atomic.StoreUint32(&c.closed, 1)
	}
	return
}
