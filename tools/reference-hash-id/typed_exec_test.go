package hashid

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestTypedTypeScriptVectors compiles the host-neutral TypeScript reference
// and executes it against the same shared Golden fixture as Go.
// This intentionally does not fetch TypeScript through a network install.
func TestTypedTypeScriptVectors(t *testing.T) {
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		t.Skip("TypeScript compiler unavailable: typed reference not verified on this host")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js unavailable: typed reference not verified on this host")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	outDir := t.TempDir()
	compile := exec.CommandContext(ctx, tsc,
		"--target", "ES2022", "--module", "NodeNext", "--strict",
		"--outDir", outDir, "hashid.mts")
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("TypeScript typecheck/compile failed: %v\n%s", err, output)
	}
	run := exec.CommandContext(ctx, node, "--test", "hashid.typed.test.mjs")
	run.Env = append(os.Environ(), "TEI_HASHID_TS_MODULE="+filepath.Join(outDir, "hashid.mjs"))
	if output, err := run.CombinedOutput(); err != nil {
		t.Fatalf("typed TypeScript cross-runtime Golden mismatch: %v\n%s", err, output)
	}
}
