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
