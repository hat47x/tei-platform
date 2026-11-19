//go:build windows && !tei_order_only && !tei_repair_only

package main

import "testing"

// The published guide requires a POSIX shell. Its literal go-run invocation
// is verified by Linux PostgreSQL/Chrome CI, not emulated via PowerShell.
func startDocumentedGuideWeb(t *testing.T, script, directory string, environment []string) <-chan error {
	t.Helper()
	t.Skip("step 5's documented POSIX shell is verified on Linux")
	return nil
}
