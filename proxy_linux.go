package netx

import (
	"net"
	"os"
	"syscall"
	"unsafe"
)

func originalTargetAddr(conn net.Conn) (n *net.TCPAddr, err error) {
	const (
		SO_ORIGINAL_DST = 80 // missing from the syscall package
	)

	type fileConn interface {
		File() (*os.File, error)
	}

	// Calling conn.File will put the socket in blocking mode, we make sure to
	// set it back to non-blocking before returning to prevent the runtime from
	// creating tons of threads because it deals with blocking syscalls all over
	// the place.
	var f *os.File
	if f, err = conn.(fileConn).File(); err != nil {
		return
	}
	defer f.Close()
	defer syscall.SetNonblock(int(f.Fd()), true)

	sock := f.Fd()
	addr := syscall.RawSockaddrAny{}
	size := uint32(unsafe.Sizeof(addr))

	_, _, e := syscall.RawSyscall6(
		uintptr(syscall.SYS_GETSOCKOPT),
		uintptr(sock),
		uintptr(syscall.SOL_IP),
		uintptr(SO_ORIGINAL_DST),
		uintptr(unsafe.Pointer(&addr)),
		uintptr(unsafe.Pointer(&size)),
		uintptr(0),
	)

	if e != 0 {
		err = e
		return
	}

	switch addr.Addr.Family {
	case syscall.AF_INET:
		a := (*syscall.RawSockaddrInet4)(unsafe.Pointer(&addr))
		n = &net.TCPAddr{
			IP:   net.IP(a.Addr[:]),
			Port: int((a.Port >> 8) | (a.Port << 8)),
		}

	case syscall.AF_INET6:
		a := (*syscall.RawSockaddrInet6)(unsafe.Pointer(&addr))
		n = &net.TCPAddr{
			IP:   net.IP(a.Addr[:]),
			Port: int((a.Port >> 8) | (a.Port << 8)),
		}

	default:
		err = syscall.EAFNOSUPPORT
	}

	return
}
