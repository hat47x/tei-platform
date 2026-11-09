package webcomposition

import (
	"os/exec"
	"strings"
	"testing"
)

// TestE1NodePlanContracts makes the read-only Node conformance tests part of
// the normal Go CI path when Node.js is installed on the runner. This checks
// the real JavaScript module loader and node:test assertions, not V8 shims.
// Go-only environments may still run the Go reference without Node; a skip
// is explicit and must not be reported as Node conformance success.
func TestE1NodePlanContracts(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is unavailable: Web E1 Node conformance was not run")
	}
	cmd := exec.Command(node, "--test", "plan-check.test.mjs", "route-table.test.mjs")
	// "go test" starts each package in its package directory.
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Web E1 Node conformance failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
}
