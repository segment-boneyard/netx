package main

import (
	"flag"
	"log"

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
	netx.ListenAndServe(bind, handler)
}
