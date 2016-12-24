package netx

import (
	"net"
	"syscall"
	"unsafe"
)

func originalTargetAddr(sock uintptr) (n *net.TCPAddr, err error) {
	const (
		SO_ORIGINAL_DST = 80 // missing from the syscall package
	)

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
