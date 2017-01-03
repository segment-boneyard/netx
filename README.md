netx [![CircleCI](https://circleci.com/gh/segmentio/netx.svg?style=shield)](https://circleci.com/gh/segmentio/netx) [![Go Report Card](https://goreportcard.com/badge/github.com/segmentio/netx)](https://goreportcard.com/report/github.com/segmentio/netx) [![GoDoc](https://godoc.org/github.com/segmentio/netx?status.svg)](https://godoc.org/github.com/segmentio/netx)
====

Go package augmenting the standard net package with more basic building blocks
for writing network applications.

Motivations
-----------

The intent of this package is to provide reusable tools that fit well with the
standard net package and extend it with features that aren't available, like a
different server implementations that have support for graceful shutdowns, and
defining interfaces that allow other packages to provide plugins that work with
these tools.
