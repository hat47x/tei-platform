package main

import (
	"runtime"
	"testing"
)

// requirePOSIXInodeReplacement skips the tests that prove a replacement is
// detected by comparing file identity. Those tests rename an existing object
// over the path it occupies, which is the POSIX way to swap one inode for
// another while keeping the same bytes.
//
// Windows has no equivalent operation to express: os.Rename onto an existing
// read-only file fails with "Access is denied" (the store writes objects 0444),
// and a directory's file identity is resolved lazily, so a replaced algorithm
// directory is not distinguishable through os.SameFile in the way these tests
// assert. The Linux CI run exercises the real check (#3460).
func requirePOSIXInodeReplacement(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX inode replacement is not expressible on Windows; the Linux CI run checks it (#3460)")
	}
}
