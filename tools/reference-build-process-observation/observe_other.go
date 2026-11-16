//go:build !(linux && amd64)

package main

import (
	"fmt"
	"os"
)

// observe is only implemented where this tool can attach with ptrace. On any
// other host it refuses instead of writing a record that looks like one.
func observe(args []string) int {
	fmt.Fprintln(os.Stderr, "observe is implemented for linux/amd64 only; no observation was made")
	return 125
}
