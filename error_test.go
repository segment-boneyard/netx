package netx

import (
	"errors"
	"testing"
)

type testError struct {
	temporary bool
	timeout   bool
}

func (e testError) Error() string   { return "test error" }
func (e testError) Temporary() bool { return e.temporary }
func (e testError) Timeout() bool   { return e.timeout }

func TestIsTemporary(t *testing.T) {
	tests := []struct {
		e error
		x bool
	}{
		{testError{temporary: false}, false},
		{testError{temporary: true}, true},
		{errors.New(""), false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if x := IsTemporary(test.e); x != test.x {
				t.Error(test.e)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		e error
		x bool
	}{
		{testError{timeout: false}, false},
		{testError{timeout: true}, true},
		{errors.New(""), false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if x := IsTimeout(test.e); x != test.x {
				t.Error(test.e)
			}
		})
	}

}

func TestTimeout(t *testing.T) {
	err := Timeout("something went wrong")

	if !IsTimeout(err) {
		t.Error("not a timeout error")
	}

	if !IsTemporary(err) {
		t.Error("not a temporary error")
	}

	if s := err.Error(); s != "something went wrong" {
		t.Error("bad error message:", s)
	}
}
