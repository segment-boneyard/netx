// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//package httplex
package httpx

import (
	"fmt"
	"testing"
)

func TestHeaderValuesContainsToken(t *testing.T) {
	tests := []struct {
		vals  []string
		token string
		want  bool
	}{
		{
			vals:  []string{"foo"},
			token: "foo",
			want:  true,
		},
		{
			vals:  []string{"bar", "foo"},
			token: "foo",
			want:  true,
		},
		{
			vals:  []string{"foo"},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"foo"},
			token: "bar",
			want:  false,
		},
		{
			vals:  []string{" foo "},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"foo,bar"},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"bar,foo,bar"},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"bar , foo"},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"foo ,bar "},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"bar, foo ,bar"},
			token: "FOO",
			want:  true,
		},
		{
			vals:  []string{"bar , foo"},
			token: "FOO",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.vals), func(t *testing.T) {
			got := headerValuesContainsToken(tt.vals, tt.token)
			if got != tt.want {
				t.Errorf("headerValuesContainsToken(%q, %q) = %v; want %v", tt.vals, tt.token, got, tt.want)
			}
		})
	}
}

func TestTokenEqual(t *testing.T) {
	tests := []struct {
		t1 string
		t2 string
		eq bool
	}{
		{
			t1: "",
			t2: "",
			eq: true,
		},
		{
			t1: "A",
			t2: "B",
			eq: false,
		},
		{
			t1: "A",
			t2: "a",
			eq: true,
		},
		{
			t1: "你好",
			t2: "你好",
			eq: false,
		},
		{
			t1: "123",
			t2: "A",
			eq: false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%q==%q:%v", test.t1, test.t2, test.eq), func(t *testing.T) {
			if eq := tokenEqual(test.t1, test.t2); eq != test.eq {
				t.Error(eq)
			}
		})
	}
}
