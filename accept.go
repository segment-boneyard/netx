package netx

import (
	"net"
	"time"
)

// Accept is a wrapper around the listener's Accept method that automatically
// handles temporary errors.
//
// An optional callback can be passed to the function to receive the temporary
// errors that are handled by the function, this can be useful to log the errors
// for example. The second argument is the amount of time the function will wait
// before retrying to accept another connection.
func Accept(lstn net.Listener, errf func(error, time.Duration)) (conn net.Conn, err error) {
	const maxBackoff = 1 * time.Second

	for attempt := 0; true; attempt++ {
		if conn, err = lstn.Accept(); err == nil {
			break
		}

		if !IsTemporary(err) {
			break
		}

		// Backoff strategy for handling temporary errors, this prevents from
		// retrying too fast when errors like running out of file descriptors
		// occur.
		backoff := time.Duration(attempt*attempt) * 10 * time.Millisecond
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		if errf != nil {
			errf(err, backoff)
		}
		time.Sleep(backoff)
	}

	return
}
