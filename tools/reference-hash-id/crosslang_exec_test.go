package hashid

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestNodeCrossRuntimeVectors(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("Node.js required for cross-runtime verification on GitHub Actions")
		}
		t.Skip("Node.js unavailable; cross-runtime comparison not run")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--test", "hashid.crosslang.test.mjs")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Node cross-runtime vectors failed: %v\n%s", err, output)
	}
}
