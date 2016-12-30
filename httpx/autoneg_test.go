package httpx

/*
HTTP Content-Type Autonegotiation.

The functions in this package implement the behaviour specified in
http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html

Copyright (c) 2011, Open Knowledge Foundation Ltd.
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

    Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.

    Redistributions in binary form must reproduce the above copyright
    notice, this list of conditions and the following disclaimer in
    the documentation and/or other materials provided with the
    distribution.

    Neither the name of the Open Knowledge Foundation Ltd. nor the
    names of its contributors may be used to endorse or promote
    products derived from this software without specific prior written
    permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
// package goautoneg

import (
	"testing"
)

const chrome = "application/xml,application/xhtml+xml,text/html;q=0.9,text/plain;q=0.8,image/png,*/*;q=0.5"

func TestNegotiate(t *testing.T) {
	tests := []struct {
		header       string
		alternatives []string
		contentType  string
	}{
		{
			header:       chrome,
			alternatives: nil,
			contentType:  "",
		},
		{
			header:       chrome,
			alternatives: []string{"text/html", "image/png"},
			contentType:  "image/png",
		},
		{
			header:       chrome,
			alternatives: []string{"text/html", "text/plain", "text/n3"},
			contentType:  "text/html",
		},
		{
			header:       chrome,
			alternatives: []string{"text/n3", "text/plain"},
			contentType:  "text/plain",
		},
		{
			header:       chrome,
			alternatives: []string{"text/n3", "application/rdf+xml"},
			contentType:  "text/n3",
		},
		{
			header:       "image/*",
			alternatives: []string{"text/plain", "image/png"},
			contentType:  "image/png",
		},
		{
			header:       "*",
			alternatives: []string{"text/plain", "image/png"},
			contentType:  "text/plain",
		},
		{
			header:       "weird/content/type",
			alternatives: []string{"text/plain", "image/png"},
			contentType:  "",
		},
	}

	for _, test := range tests {
		if contentType := Negotiate(test.header, test.alternatives); contentType != test.contentType {
			t.Errorf("%s %v: %s != %s", test.header, test.alternatives, test.contentType, contentType)
		}
	}
}
