package httpx

import (
	"io"
	"sync"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} { return &buffer{make([]byte, 16384)} },
	}
)

type buffer struct {
	b []byte
}

func getBuffer() *buffer {
	return bufferPool.Get().(*buffer)
}

func putBuffer(b *buffer) {
	bufferPool.Put(b)
}

func bufferedCopy(w io.Writer, r io.Reader) (n int64, err error) {
	buf := getBuffer()
	n, err = io.CopyBuffer(w, r, buf.b)
	putBuffer(buf)
	return
}
