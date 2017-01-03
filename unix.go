package netx

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
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
	var c = conn.(fileConn)
	var f *os.File

	if f, err = c.File(); err != nil {
		return
	}
	defer f.Close()

	if err = SendUnixFile(socket, f); err != nil {
		return
	}

	conn.Close()
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

// RecvUnixFile receives a file descriptor from a unix domain socket.
func RecvUnixFile(socket *net.UnixConn) (file *os.File, err error) {
	var oob = make([]byte, syscall.CmsgSpace(4))
	var msg []syscall.SocketControlMessage
	var fds []int

	if _, _, _, _, err = socket.ReadMsgUnix(nil, oob); err != nil {
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

// RecvUnixListener returns a new listener which accepts connection by reading
// file descriptors from a unix domain socket.
func RecvUnixListener(socket *net.UnixConn) net.Listener {
	return recvUnixListener{socket}
}

type recvUnixListener struct {
	socket *net.UnixConn
}

func (l recvUnixListener) Accept() (net.Conn, error) {
	return RecvUnixConn(l.socket)
}

func (l recvUnixListener) Addr() net.Addr {
	return l.socket.LocalAddr()
}

func (l recvUnixListener) Close() error {
	return l.socket.Close()
}

// SendUnixHandler wraps handler so the connetions it receives will be sent back
// to socket when they are closed.
func SendUnixHandler(socket *net.UnixConn, handler Handler) Handler {
	return sendUnixHandler{
		handler: handler,
		socket:  socket,
	}
}

type sendUnixHandler struct {
	handler Handler
	socket  *net.UnixConn
}

func (h sendUnixHandler) ServeConn(ctx context.Context, conn net.Conn) {
	c := &sendUnixConn{Conn: conn}
	defer c.Close()
	h.handler.ServeConn(ctx, c)
}

type sendUnixConn struct {
	net.Conn
	once   sync.Once
	socket *net.UnixConn
}

func (c *sendUnixConn) Close() (err error) {
	c.once.Do(func() {
		if err = SendUnixConn(c.socket, c.Conn); err != nil {
			c.Conn.Close()
		}
	})
	return
}
