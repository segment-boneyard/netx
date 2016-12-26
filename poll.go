package netx

import "net"

// PollRead asynchronously waits for conn to become ready, closing the ready
// channel when the event occurs.
// The cancel function should be called to unregister internal resources if the
// operation needs to be aborted.
func PollRead(conn net.Conn) (ready <-chan struct{}, cancel func(), err error) {
	return pollRead(conn)
}
