package httpx

import (
	"bufio"
	"context"
	"net"
	"os"
	"time"

	"github.com/segmentio/netx"
)

// pollRead implements a polling that doesn't rely on netx.PollRead and then
// works in cases where the net.Conn instances don't implement netx.File (it
// is less efficient than using PollRead tho).
func pollRead(ctx context.Context, conn net.Conn, r *bufio.Reader, timeout time.Duration) (err error) {
	deadline := time.Now().Add(timeout)
	done := ctx.Done()

	for {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		if _, err = r.Peek(1); err == nil {
			return
		}

		if !netx.IsTimeout(err) {
			return
		}

		if deadline.After(time.Now()) {
			err = netx.Timeout("i/o timeout waiting for an HTTP request")
			return
		}

		select {
		case <-done:
			return
		default:
		}
	}
}

// waitRead waits for data to become available on f, potentially aborting the
// operation if timeout expires or ctx is canceled.
func waitRead(ctx context.Context, f *os.File, timeout time.Duration) (err error) {
	var timec <-chan time.Time
	var ready <-chan struct{}
	var cancel func()

	if timeout != 0 {
		timer := time.NewTimer(timeout)
		timec = timer.C
		defer timer.Stop()
	}

	if ready, cancel, err = netx.PollRead(f); err != nil {
		return
	}
	defer cancel()

	select {
	case <-ready:
	case <-timec:
		err = netx.Timeout("i/o timeout waiting for an HTTP request")
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}
