package netx

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

func TestCopy(t *testing.T) {
	t.Run("WriterTo", func(t *testing.T) {
		w := bytes.NewBuffer(nil)
		r := bytes.NewBuffer([]byte("Hello World!"))

		if n, err := Copy(w, r); err != nil {
			t.Error(err)
		} else if n != 12 {
			t.Error("bad byte count:", n)
		}

		if s := w.String(); s != "Hello World!" {
			t.Error("bad output:", s)
		}
	})

	t.Run("ReaderFrom", func(t *testing.T) {
		w := bytes.NewBuffer(nil)
		r := &testBuffer{[]byte("Hello World!")}

		if n, err := Copy(w, r); err != nil {
			t.Error(err)
		} else if n != 12 {
			t.Error("bad byte count:", n)
		}

		if s := w.String(); s != "Hello World!" {
			t.Error("bad output:", s)
		}
	})

	t.Run("Basic", func(t *testing.T) {
		c1, c2, err := ConnPair("tcp")
		if err != nil {
			t.Error(err)
			return
		}
		defer c1.Close()
		defer c2.Close()

		go Copy(c1, c2)

		if _, err := c2.Write([]byte("Hello World!")); err != nil {
			t.Error(err)
			return
		}
		c2.Close()

		b, err := ioutil.ReadAll(c1)
		if err != nil {
			t.Error(err)
			return
		}

		if s := string(b); s != "Hello World!" {
			t.Error("bad output:", s)
		}
	})
}

type testBuffer struct{ b []byte }

func (buf *testBuffer) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return
	}
	if len(buf.b) == 0 {
		err = io.EOF
		return
	}
	n = copy(b, buf.b)
	buf.b = buf.b[n:]
	return
}
