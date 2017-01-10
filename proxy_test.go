package netx

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"testing"
)

type readOneByOne struct {
	b []byte
}

func (r *readOneByOne) Read(b []byte) (n int, err error) {
	if len(r.b) == 0 {
		err = io.EOF
	} else if len(b) != 0 {
		n, b[0], r.b = 1, r.b[0], r.b[1:]
	}
	return
}

func TestProxyProtoV1(t *testing.T) {
	tests := []struct {
		src net.Addr
		dst net.Addr
		str string
	}{
		{
			src: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 56789},
			dst: &net.TCPAddr{IP: net.ParseIP("192.1.0.123"), Port: 4242},
			str: "PROXY TCP4 127.0.0.1 192.1.0.123 56789 4242\r\n",
		},
		{
			src: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 56789},
			dst: &net.TCPAddr{IP: net.ParseIP("fe80::f65c:89ff:feac:29cb"), Port: 4242},
			str: "PROXY TCP6 ::1 fe80::f65c:89ff:feac:29cb 56789 4242\r\n",
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s->%s", test.src, test.dst), func(t *testing.T) {
			b := appendProxyProtoV1(nil, test.src, test.dst)

			if s := string(b); s != test.str {
				t.Error("bad proxy proto header:", s)
				return
			}

			r := &readOneByOne{b}
			a1, a2, buf, local, err := parseProxyProto(r)

			if err != nil {
				t.Error(err)
			}

			if len(r.b) != 0 || len(buf) != 0 {
				t.Error("unexpected trailing bytes")
			}

			if !reflect.DeepEqual(test.src, a1) {
				t.Errorf("bad source: %#v", a1)
			}

			if !reflect.DeepEqual(test.dst, a2) {
				t.Errorf("bad destination: %#v", a2)
			}

			if local {
				t.Errorf("bad local: %t", local)
			}
		})
	}
}

func TestProxyProtoV2(t *testing.T) {
	tests := []struct {
		src net.Addr
		dst net.Addr
		str string
	}{
		{
			src: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 56789},
			dst: &net.TCPAddr{IP: net.ParseIP("192.1.0.123"), Port: 4242},
		},
		{
			src: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 56789},
			dst: &net.TCPAddr{IP: net.ParseIP("fe80::f65c:89ff:feac:29cb"), Port: 4242},
		},
		{
			src: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 56789},
			dst: &net.UDPAddr{IP: net.ParseIP("192.1.0.123"), Port: 4242},
		},
		{
			src: &net.UDPAddr{IP: net.ParseIP("::1"), Port: 56789},
			dst: &net.UDPAddr{IP: net.ParseIP("fe80::f65c:89ff:feac:29cb"), Port: 4242},
		},
		{
			src: &net.UnixAddr{Net: "unix", Name: "/var/lib/src.sock"},
			dst: &net.UnixAddr{Net: "unix", Name: "/var/lib/dst.sock"},
		},
		{
			src: &net.UnixAddr{Net: "unixgram", Name: "/var/lib/src.sock"},
			dst: &net.UnixAddr{Net: "unixgram", Name: "/var/lib/dst.sock"},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s://%s->%s", test.src.Network(), test.src, test.dst), func(t *testing.T) {
			b := appendProxyProtoV2(nil, test.src, test.dst, false)
			r := &readOneByOne{b}
			a1, a2, buf, local, err := parseProxyProto(r)

			if err != nil {
				t.Error(err)
			}

			if len(r.b) != 0 || len(buf) != 0 {
				t.Errorf("unexpected trailing bytes: %#v %#v", r.b, buf)
			}

			if !reflect.DeepEqual(test.src, a1) {
				t.Errorf("bad source: %#v", a1)
			}

			if !reflect.DeepEqual(test.dst, a2) {
				t.Errorf("bad destination: %#v", a2)
			}

			if local {
				t.Errorf("bad local state: %t", local)
			}
		})
	}
}

func TestProxyProtoV2Local(t *testing.T) {
	b := appendProxyProtoV2(nil, &NetAddr{}, &NetAddr{}, true)
	r := &readOneByOne{b}
	src, dst, buf, local, err := parseProxyProto(r)

	if err != nil {
		t.Error(err)
	}

	if len(r.b) != 0 || len(buf) != 0 {
		t.Errorf("unexpected trailing bytes: %#v %#v", r.b, buf)
	}

	if src != nil {
		t.Errorf("bad source: %#v", src)
	}

	if dst != nil {
		t.Errorf("bad destination: %#v", dst)
	}

	if !local {
		t.Errorf("bad local state: %t", local)
	}
}
