package netx

import "testing"

func TestNetAddr(t *testing.T) {
	a := &NetAddr{
		Net:  "N",
		Addr: "A",
	}

	if s := a.Network(); s != "N" {
		t.Error("bad network:", s)
	}

	if s := a.String(); s != "A" {
		t.Error("bad address:", s)
	}
}

func TestMultiAddr(t *testing.T) {
	m := MultiAddr{
		&NetAddr{"N1", "A1"},
		&NetAddr{"N2", "A2"},
	}

	if s := m.Network(); s != "N1,N2" {
		t.Error("bad network:", s)
	}

	if s := m.String(); s != "A1,A2" {
		t.Error("bad address:", s)
	}
}

func TestSplitNetAddr(t *testing.T) {
	tests := []struct {
		s string
		n string
		a string
	}{
		{
			s: "",
			n: "",
			a: "",
		},
		{
			s: "tcp://",
			n: "tcp",
			a: "",
		},
		{
			s: "127.0.0.1:4242",
			n: "",
			a: "127.0.0.1:4242",
		},
		{
			s: "tcp://127.0.0.1:4242",
			n: "tcp",
			a: "127.0.0.1:4242",
		},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			n, a := SplitNetAddr(test.s)

			if n != test.n {
				t.Error("bad network:", n)
			}

			if a != test.a {
				t.Error("bad address:", a)
			}
		})
	}
}

func TestSplitAddrPort(t *testing.T) {
	tests := []struct {
		s string
		a string
		p int
	}{
		{
			s: "",
			a: "",
			p: -1,
		},
		{
			s: "127.0.0.1",
			a: "127.0.0.1",
			p: -1,
		},
		{
			s: "127.0.0.1:4242",
			a: "127.0.0.1",
			p: 4242,
		},
		{
			s: "[::1]:4242",
			a: "::1",
			p: 4242,
		},
		{
			s: ":1234",
			a: "",
			p: 1234,
		},
		{
			s: "127.0.0.1:http",
			a: "127.0.0.1",
			p: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			a, p := SplitAddrPort(test.s)

			if a != test.a {
				t.Error("bad address:", a)
			}

			if p != test.p {
				t.Error("bad port:", p)
			}
		})
	}
}
