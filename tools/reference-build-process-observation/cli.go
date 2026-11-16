// Command reference-build-process-observation keeps three accounts of one
// build apart: what was declared, what the build reported about itself, and
// what the operating system saw.
//
//	observe    run a command under ptrace and record every program image that
//	           was executed in its process tree and, with -files, every path
//	           that was opened (linux/amd64 only)
//	scope      turn a `tei self-build` report into the declared scope: the
//	           images, toolchain root and path roots the attempt named
//	reconcile  compare an observation with a declared scope and say which
//	           executions and opens the declaration covers, which it does not,
//	           and what the observation could not see
//
// It is an observation instrument, not a sandbox and not a build engine. It
// starts no build step of its own and decides nothing about acceptance.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "observe":
		os.Exit(observe(os.Args[2:]))
	case "scope":
		os.Exit(scopeCommand(os.Args[2:]))
	case "reconcile":
		os.Exit(reconcileCommand(os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: reference-build-process-observation observe|scope|reconcile [flags]")
}
