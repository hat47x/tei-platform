package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Exercise the existing immutable archive verifier through the POC-12
// reconciliation tests in the normal Go repository-health CI on hosts with
// Python 3. No Source snapshots or GitHub credentials are inferred.
func TestE1ArchiveReconciliationOffline(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("optional Python 3 archive inspection is unavailable on this host")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-B",
		"tools/reference-change-assurance/test_e1_archive_reconcile.py")
	cmd.Dir = filepath.Join("..", "..")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("existing E1 archive reconciliation regressions failed: %v\n%s", err, out)
	}
}
