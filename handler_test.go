package netx

import (
	"context"
	"io/ioutil"
	"testing"
	"time"
)

func TestEcho(t *testing.T) {
	c1, c2, err := TCPConnPair("tcp")
	if err != nil {
		t.Error(err)
		return
	}
	defer c1.Close()
	defer c2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go CloseHandler(Echo).ServeConn(ctx, c2)

	if _, err := c1.Write([]byte("Hello World!\n")); err != nil {
		t.Error(err)
		return
	}
	c1.CloseWrite()

	b, err := ioutil.ReadAll(c1)
	if err != nil {
		t.Error(err)
	}
	if s := string(b); s != "Hello World!\n" {
		t.Error("bad output:", s)
	}
}

func TestEchoLine(t *testing.T) {
	c1, c2, err := TCPConnPair("tcp")
	if err != nil {
		t.Error(err)
		return
	}
	defer c1.Close()
	defer c2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go CloseHandler(EchoLine).ServeConn(ctx, c2)

	if _, err := c1.Write([]byte("Hello World!\r\n")); err != nil {
		t.Error(err)
		return
	}
	c1.CloseWrite()

	b, err := ioutil.ReadAll(c1)
	if err != nil {
		t.Error(err)
	}
	if s := string(b); s != "Hello World!\r\n" {
		t.Error("bad output:", s)
	}
}

func TestPass(t *testing.T) {
	c1, c2, err := TCPConnPair("tcp")
	if err != nil {
		t.Error(err)
		return
	}
	defer c1.Close()
	defer c2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go CloseHandler(Pass).ServeConn(ctx, c2)

	b, err := ioutil.ReadAll(c1)
	if err != nil {
		t.Error(err)
	}
	if s := string(b); s != "" {
		t.Error("bad output:", s)
	}
}
