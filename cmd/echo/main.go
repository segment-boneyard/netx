package main

import (
	"flag"
	"log"

	"github.com/segmentio/netx"
)

func main() {
	var bind string

	flag.StringVar(&bind, "bind", ":4242", "The network address to listen for incoming connections.")
	flag.Parse()

	log.Printf("listening on %s", bind)
	netx.ListenAndServe(bind, netx.Echo)
}
