package netx

import (
	"os"
	"syscall"
)

func socketpair(domain int, socktype int, protocol int) (fd1 int, fd2 int, err error) {
	var fds [2]int
	fd1 = -1
	fd2 = -1

	if fds, err = syscall.Socketpair(domain, socktype|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK, 0); err != nil {
		err = os.NewSyscallError("socketpair", err)
		return
	}

	fd1 = fds[0]
	fd2 = fds[1]
	return
}
