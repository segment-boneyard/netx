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
