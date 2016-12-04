netx
====

Go package augmenting the standard net package with more basic building blocks
for writing network applications.

Motivations
-----------

The intent of this package is to provide reusable tools that fit well with the
stadnard net package and extend it with features that aren't available, like a
different server implementations that have support for graceful shutdowns, and
defining interfaces that allow other packages to provide plugins that work with
these tools.
