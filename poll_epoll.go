// +build linux

package netx

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

type filePoller struct {
	fd    uintptr
	once  sync.Once
	mutex sync.Mutex
	files map[uintptr](chan<- struct{})
}

func (p *filePoller) init() {
	p.once.Do(func() {
		fd, err := epollCreate()
		if err != nil {
			panic(err)
		}

		p.fd = fd
		p.files = make(map[uintptr](chan<- struct{}))

		go func(p *filePoller) {
			// Lock the OS thread because we're using blocking syscalls on this
			// goroutine.
			runtime.LockOSThread()

			// Run the event queue loop, polling for events and then dispatching
			// them to the registered files.
			var events [128]syscall.EpollEvent
			for {
				n, err := syscall.EpollWait(int(p.fd), events[:], -1)

				switch err {
				case nil:
				case syscall.EINTR:
				case syscall.EBADF:
					return // the poller was closed
				default:
					panic(err)
				}

				if n <= 0 {
					continue
				}
				p.mutex.Lock()

				for _, ev := range events[:n] {
					fd := uintptr(ev.Fd)
					ch := p.files[fd]
					delete(p.files, fd)

					if ch != nil {
						close(ch)
					}
				}

				p.mutex.Unlock()
			}
		}(p)
	})
	return
}

func (p *filePoller) register(f *os.File) (ready <-chan struct{}, cancel func(), err error) {
	p.init()

	ch := make(chan struct{})
	fd := f.Fd()

	p.mutex.Lock()
	p.files[fd] = ch
	p.mutex.Unlock()

	if err = epollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Fd:     int32(fd),
		Events: syscall.EPOLLIN | syscall.EPOLLPRI | syscall.EPOLLHUP | syscall.EPOLLRDHUP | syscall.EPOLLONESHOT,
	}); err != nil {
		p.mutex.Lock()
		delete(p.files, fd)
		p.mutex.Unlock()
		return
	}

	cancel = func() {
		epollCtl(p.fd, syscall.EPOLL_CTL_DEL, fd, nil)
		p.mutex.Lock()
		delete(p.files, fd)
		p.mutex.Unlock()
	}

	ready = ch
	return
}

var (
	poller filePoller
)

func pollRead(f *os.File) (<-chan struct{}, func(), error) {
	return poller.register(f)
}

func errno(err syscall.Errno) error {
	if err == 0 {
		return nil
	}
	return err
}

func epollCreate() (uintptr, error) {
	r, _, e := syscall.RawSyscall(
		syscall.SYS_EPOLL_CREATE1,
		syscall.EPOLL_CLOEXEC,
		0,
		0,
	)
	return r, errno(e)
}

func epollCtl(epfd uintptr, op uintptr, fd uintptr, event *syscall.EpollEvent) error {
	_, _, e := syscall.RawSyscall6(
		syscall.SYS_EPOLL_CTL,
		epfd,
		op,
		fd,
		uintptr(unsafe.Pointer(event)),
		0,
		0,
	)
	return errno(e)
}
