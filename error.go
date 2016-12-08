package netx

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
