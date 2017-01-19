package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/segmentio/netx"
)

func main() {
	var bind string
	var mode string

	flag.StringVar(&bind, "bind", ":4242", "The network address to listen for incoming connections.")
	flag.StringVar(&mode, "mode", "raw", "The echo mode, either 'line' or 'raw'")
	flag.Parse()

	var handler netx.Handler

	switch mode {
	case "line":
		handler = netx.EchoLine
	case "raw":
		handler = netx.Echo
	default:
		log.Fatal("bad echo mode:", mode)
	}

	log.Printf("setting echo mode to '%s'", mode)
	log.Printf("listening on %s", bind)

	lstn, err := netx.Listen(bind)
	if err != nil {
		log.Fatal(err)
	}

	if u, ok := lstn.(*netx.RecvUnixListener); ok {
		c, err := netx.DupUnix(u.UnixConn())
		if err != nil {
			log.Fatal(err)
		}
		handler = netx.NewSendUnixHandler(c, handler)
	}

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-sigchan
		log.Print("signal: ", sig)
		cancel()
	}()

	server := &netx.Server{
		Handler: handler,
		Context: ctx,
	}

	if err := server.Serve(lstn); err != nil {
		log.Fatal(err)
	}
}
