package netx

import (
	"errors"
	"net"
)

// Timeout returns a new network error representing a timeout.
func Timeout(msg string) net.Error { return &timeout{msg} }

type timeout struct{ msg string }

func (t *timeout) Error() string   { return t.msg }
func (t *timeout) Timeout() bool   { return true }
func (t *timeout) Temporary() bool { return true }

// IsTemporary checks whether err is a temporary error.
func IsTemporary(err error) bool {
	e, ok := err.(interface {
		Temporary() bool
	})
	return ok && e != nil && e.Temporary()
}

// IsTimeout checks whether err resulted from a timeout.
func IsTimeout(err error) bool {
	e, ok := err.(interface {
		Timeout() bool
	})
	return ok && e != nil && e.Timeout()
}

var (
	// ErrLineTooLong should be used by line-based protocol readers that detect
	// a line longer than they were configured to handle.
	ErrLineTooLong = errors.New("the line is too long")

	// ErrNoPipeline should be used by handlers that detect an attempt to use
	// pipelining when they don't support it.
	ErrNoPipeline = errors.New("pipelining is not supported")
)
