package httpx

// =============================================================================
// This is a modified version of the the bitbucket.org/ww/goautoneg package so
// httpx has no external dependency.
// It also makes a couple of performance improvements and we've made sure to get
// 100% test coverage on the file.
// =============================================================================

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
// package autoneg

import (
	"sort"
	"strconv"
	"strings"
)

// Structure to represent a clause in an HTTP Accept Header
type accept struct {
	Type, SubType string
	Q             float64
	//	Params        map[string]string
}

// For internal use, so that we can use the sort interface
type acceptList []accept

func (s acceptList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s acceptList) Len() int { return len(s) }

func (s acceptList) Less(i, j int) bool {
	ai, aj := &s[i], &s[j]
	return (ai.Q > aj.Q) || (ai.Type != "*" && aj.Type == "*") || (ai.SubType != "*" && aj.SubType == "*")
}

// Parse an Accept Header string returning a sorted list
// of clauses
func parseAccept(header string) (acc []accept) {
	parts := strings.Split(header, ",")
	acc = make([]accept, 0, len(parts))
	for _, part := range parts {
		part := trimOWS(part)

		a := accept{}
		// Removed: the parameters werer only written and never read, so there's just
		// point building this map, it just slows down the function for no reason.
		//a.Params = make(map[string]string)
		a.Q = 1.0

		mrp := strings.Split(part, ";")

		mediaRange := mrp[0]
		sp := strings.Split(mediaRange, "/")
		a.Type = trimOWS(sp[0])

		switch {
		case len(sp) == 1 && a.Type == "*":
			a.SubType = "*"
		case len(sp) == 2:
			a.SubType = trimOWS(sp[1])
		default:
			continue
		}

		if len(mrp) == 1 {
			acc = append(acc, a)
			continue
		}

		for _, param := range mrp[1:] {
			if sp := strings.SplitN(param, "=", 2); len(sp) == 2 {
				if token := trimOWS(sp[0]); token == "q" {
					a.Q, _ = strconv.ParseFloat(sp[1], 32)
				} /*else {
					a.Params[token] = trimOWS(sp[1])
				}*/
			}
		}

		acc = append(acc, a)
	}

	sort.Sort(acceptList(acc))
	return
}

// Negotiate the most appropriate content-type given the accept header
// and a list of alternatives.
func Negotiate(header string, alternatives []string) (contentType string) {
	asp := make([][]string, 0, len(alternatives))

	for _, ctype := range alternatives {
		asp = append(asp, strings.SplitN(ctype, "/", 2))
	}

	for _, clause := range parseAccept(header) {
		for i, ctsp := range asp {
			if clause.Type == ctsp[0] && clause.SubType == ctsp[1] {
				contentType = alternatives[i]
				return
			}

			if clause.Type == ctsp[0] && clause.SubType == "*" {
				contentType = alternatives[i]
				return
			}

			if clause.Type == "*" && clause.SubType == "*" {
				contentType = alternatives[i]
				return
			}
		}
	}

	return
}
