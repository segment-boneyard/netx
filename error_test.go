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
