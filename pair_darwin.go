package netx

import (
	"os"
	"syscall"
)

func socketpair(domain int, socktype int, protocol int) (fd1 int, fd2 int, err error) {
	var fds [2]int
	fd1 = -1
	fd2 = -1
	syscall.ForkLock.Lock()

	if fds, err = syscall.Socketpair(domain, socktype, protocol); err != nil {
		err = os.NewSyscallError("socketpair", err)
	}

	syscall.CloseOnExec(fds[0])
	syscall.CloseOnExec(fds[1])
	syscall.ForkLock.Unlock()

	if err = syscall.SetNonblock(fds[0], true); err != nil {
		syscall.Close(fds[0])
		syscall.Close(fds[1])
		err = os.NewSyscallError("setnonblock", err)
		return
	}

	if err = syscall.SetNonblock(fds[1], true); err != nil {
		syscall.Close(fds[0])
		syscall.Close(fds[1])
		err = os.NewSyscallError("setnonblock", err)
		return
	}

	fd1 = fds[0]
	fd2 = fds[1]
	return
}
