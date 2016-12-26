// +build darwin

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

func (fp *file) close() {
	fp.m.Lock()

	if fp.f != nil {
		fp.f.Close()
		fp.f = nil
		close(fp.c)
	}

	fp.m.Unlock()
}

func (p *filePoller) init() {
	p.once.Do(func() {
		fd, err := kqueue()
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
			var events [128]syscall.Kevent_t
			for {
				n, err := syscall.Kevent(int(p.fd), nil, events[:], nil)

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
					fd := uintptr(ev.Ident)
					fp := p.files[fd]
					delete(p.files, fd)
					p.mutex.Unlock()

					if fp != nil {
						fp.close()
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
		fp.close()
	}

	p.mutex.Lock()
	p.files[fd] = fp
	p.mutex.Unlock()

	if err = kevent(p.fd, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_EOF | syscall.EV_CLEAR,
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

func kqueue() (uintptr, error) {
	r, _, e := syscall.RawSyscall(syscall.SYS_KQUEUE, 0, 0, 0)
	return r, errno(e)
}

func kevent(kq uintptr, ev syscall.Kevent_t) error {
	var changes = [1]syscall.Kevent_t{ev}
	var timespec syscall.Timespec // zero, don't block

	_, _, e := syscall.RawSyscall6(
		syscall.SYS_KEVENT,
		uintptr(kq),
		uintptr(unsafe.Pointer(&changes[0])),
		uintptr(1),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&timespec)),
	)

	return errno(e)
}
