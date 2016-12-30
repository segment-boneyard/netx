package netx

import (
	"net"
	"testing"
	"time"
)

func TestPollRead(t *testing.T) {
	net0, addr0, close0 := listenAndServe(&Echo{})
	defer close0()

	conn1, err := net.Dial(net0, addr0)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn1.Close()
	f1, _ := conn1.(File).File()
	defer f1.Close()

	conn2, err := net.Dial(net0, addr0)
	if err != nil {
		t.Error(err)
		return
	}
	defer conn2.Close()
	f2, _ := conn2.(File).File()
	defer f2.Close()

	ready1, cancel1, err := PollRead(f1)
	if err != nil {
		t.Error(err)
		return
	}
	defer cancel1()

	ready2, cancel2, err := PollRead(f2)
	if err != nil {
		t.Error(err)
		return
	}
	defer cancel2()

	// Make sure that receiving data triggers the ready channel.
	if _, err := conn2.Write([]byte("Hello World!")); err != nil {
		t.Error(err)
		return
	}

	select {
	case <-ready1:
		t.Error("invalid channel triggered")
	case <-ready2:
	case <-time.After(100 * time.Millisecond):
		t.Error("no channel triggered within 100ms of sending data on the connection")
	}
}
