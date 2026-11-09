package main

import (
	"encoding/json"
	"testing"
)

// childArgs encodes a child argv as the JSON array -child-arg expects. The
// tests used to build the JSON by concatenating the artifact path into a raw
// string; on Windows that path contains backslashes, and a backslash inside a
// JSON string is an escape sequence, so the CLI rejected the argument before
// starting the child (#3149 Windows survey). Encoding with encoding/json also
// keeps the path's spaces and non-ASCII characters intact.
func childArgs(t *testing.T, args ...string) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
