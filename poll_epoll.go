// +build linux

package netx

import (
	"net"
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
	files map[uintptr]*file
}

type file struct {
	m sync.Mutex
	f *os.File
	c chan<- struct{}
}

func (fp *file) close(p *filePoller) {
	fp.m.Lock()

	if fp.f != nil {
		epollCtl(p.fd, syscall.EPOLL_CTL_DEL, fp.f.Fd(), nil)
		fp.f.Close()
		fp.f = nil
		close(fp.c)
	}

	fp.m.Unlock()
}

func (p *filePoller) init() {
	p.once.Do(func() {
		fd, err := epollCreate()
		if err != nil {
			panic(err)
		}

		p.fd = fd
		p.files = make(map[uintptr]*file)

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

				for _, ev := range events[:n] {
					p.mutex.Lock()
					fd := uintptr(ev.Fd)
					fp := p.files[fd]
					delete(p.files, fd)
					p.mutex.Unlock()

					if fp != nil {
						fp.close(p)
					}
				}
			}
		}(p)
	})
	return
}

func (p *filePoller) register(conn net.Conn) (ready <-chan struct{}, cancel func(), err error) {
	p.init()

	var f *os.File

	if f, err = conn.(interface {
		File() (*os.File, error)
	}).File(); err != nil {
		return
	}

	c := make(chan struct{})
	fd := f.Fd()
	fp := &file{
		f: f,
		c: c,
	}

	cancel = func() {
		p.mutex.Lock()
		delete(p.files, fd)
		p.mutex.Unlock()
		fp.close(p)
	}

	p.mutex.Lock()
	p.files[fd] = fp
	p.mutex.Unlock()

	if err = epollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Fd:     int32(fd),
		Events: syscall.EPOLLIN | syscall.EPOLLPRI | syscall.EPOLLHUP | syscall.EPOLLRDHUP | syscall.EPOLLONESHOT,
	}); err != nil {
		cancel()
		cancel = nil
		return
	}

	ready = c
	return
}

var (
	poller filePoller
)

func pollRead(conn net.Conn) (ready <-chan struct{}, cancel func(), err error) {
	return poller.register(conn)
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
