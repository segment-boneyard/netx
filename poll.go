package netx

import "os"

// File is an interface that is implemented by connections that can be converted
// to files.
type File interface {
	// File returns an os.File representing the same operating system resource.
	File() (*os.File, error)
}

// PollRead asynchronously waits for f to become ready for reading, writing to
// the ready channel when the event occurs.
// The cancel function should be called to unregister internal resources if the
// operation needs to be aborted.
//
// Calling PollRead on the same file concurrently from multiple goroutine is not
// supported and will lead to undefined behaviors.
//
// Asynchronously closing the file will not trigger the ready channel (since no
// data can be read after the file is closed), the application should use other
// synchronization mechanisms to be notified of a file being closed. However,
// other "closing" events like the file reaching EOF will trigger a ready event.
func PollRead(f *os.File) (ready <-chan struct{}, cancel func(), err error) {
	return pollRead(f)
}
