package netx

// IsTemporary checks whether err is a temporary error.
func IsTemporary(err error) bool {
	if err != nil {
		if e, ok := err.(interface {
			Temporary() bool
		}); ok {
			return e.Temporary()
		}
	}
	return false
}

// IsTimeout checks whether err resulted from a timeout.
func IsTimeout(err error) bool {
	if err != nil {
		if e, ok := err.(interface {
			Timeout() bool
		}); ok {
			return e.Timeout()
		}
	}
	return false
}
