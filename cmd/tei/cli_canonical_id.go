package main

import (
	"fmt"
	"os"

	"tei/internal/core"
)

// canonicalOrExit parses one user-supplied identifier, or ends the command
// with exit code 2 and the parse rule on standard error.
//
// The commands below take IDs straight from flags. Calling MustCanonicalID on
// that text turns a mistyped identifier into a panic: a stack trace and exit
// status 2, which reads like a program fault rather than "this argument is
// wrong". `tei desired-schema -database sales-core` (a hyphen is not legal in a
// canonical segment) printed a goroutine dump for what is only a bad argument.
//
// The message names the flag and repeats the parser's rule, so a caller can fix
// the value without reading the source.
func canonicalOrExit(flag, value string) core.CanonicalID {
	id, err := core.ParseCanonicalID(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid -%s: %v\n", flag, err)
		os.Exit(2)
	}
	return id
}
