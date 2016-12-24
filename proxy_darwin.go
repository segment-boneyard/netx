package netx

import (
	"net"
	"syscall"
)

func originalTargetAddr(sock uintptr) (*net.TCPAddr, error) {
	return nil, syscall.ENOSYS
}
