package netx

import "testing"

func TestIsIP(t *testing.T) {
	tests := []struct {
		s string
		x bool
	}{
		{"", false},
		{"127", false},
		{"127.0.0.1", true},
		{"::", true},
		{"::1", true},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if x := IsIP(test.s); x != test.x {
				t.Error(test.s)
			}
		})
	}
}

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		s string
		x bool
	}{
		{"", false},
		{"127", false},
		{"127.0.0.1", true},
		{"::", false},
		{"::1", false},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if x := IsIPv4(test.s); x != test.x {
				t.Error(test.s)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		s string
		x bool
	}{
		{"", false},
		{"127", false},
		{"127.0.0.1", false},
		{"::", true},
		{"::1", true},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if x := IsIPv6(test.s); x != test.x {
				t.Error(test.s)
			}
		})
	}
}
