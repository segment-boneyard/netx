package netx

import (
	"io"
	"sync"
)

// Copy behaves exactly like io.Copy but uses an internal buffer pool to release
// pressure off of the garbage collector.
func Copy(w io.Writer, r io.Reader) (n int64, err error) {
	if from, ok := r.(io.WriterTo); ok {
		return from.WriteTo(w)
	}

	if to, ok := w.(io.ReaderFrom); ok {
		return to.ReadFrom(r)
	}

	buf := bufferPool.Get().(*buffer)
	n, err = io.CopyBuffer(w, r, buf.b)
	bufferPool.Put(buf)
	return
}

// buffer is a simple wrapper around []byte, it prevents Go from making a memory
// allocation when converting the byte slice to an interface{}.
type buffer struct{ b []byte }

var (
	bufferPool = sync.Pool{
		New: func() interface{} { return &buffer{make([]byte, 8192, 8192)} },
	}
)
