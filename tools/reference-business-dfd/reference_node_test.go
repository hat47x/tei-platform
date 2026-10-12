package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBusinessDFDReferenceNodeSuite runs the actual Node reference tests, not
// a reconstructed V8 harness. Go-only environments may omit Node; the test
// explicitly records that those checks were not exercised.
func TestBusinessDFDReferenceNodeSuite(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("DFD JavaScript reference tests not executed: Node.js is unavailable")
	}

	root := filepath.Join("..", "..", "reference", "business-system-design")
	suites := []string{
		"dfd-edit-proposal.test.mjs",
		"dfd-edit-boundary.test.mjs",
		"dfd-edit-cli.test.mjs",
		"dfd-selected-edit-bridge.test.mjs",
		"dfd-selected-edit-cli.test.mjs",
		"dfd-view-layout.test.mjs",
		"dfd-view-layout-cli.test.mjs",
		"dfd-view-studio.test.mjs",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	args := append([]string{"--test"}, suites...)
	cmd := exec.CommandContext(ctx, node, args...)
	cmd.Dir = root
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("DFD Node tests exceeded timeout: %v\n%s", ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("DFD Node tests failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "# fail 0") {
		t.Fatalf("DFD Node runner did not report zero failures:\n%s", output)
	}
	t.Logf("DFD Node reference tests completed using %s\n%s", node, output)
}
