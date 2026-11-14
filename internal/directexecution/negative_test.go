package directexecution_test

import (
	"os"
	"strings"
	"testing"

	"tei/internal/directexecution"
)

func TestRunRejectsMissingOutput(t *testing.T) {
	if _, err := directexecution.Run(workerOptions(t, "no-output")); err == nil {
		t.Fatal("expected missing output error")
	}
}
func TestRunRejectsNonZeroExit(t *testing.T) {
	if _, err := directexecution.Run(workerOptions(t, "fail")); err == nil {
		t.Fatal("expected execution error")
	}
}
func TestRunRejectsPreexistingOutput(t *testing.T) {
	opts := workerOptions(t, "write")
	if err := os.WriteFile(opts.OutputPath, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := directexecution.Run(opts); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected preexisting output error: %v", err)
	}
	b, err := os.ReadFile(opts.OutputPath)
	if err != nil || string(b) != "old" {
		t.Fatalf("preexisting output changed: %q %v", b, err)
	}
}
func TestRunRejectsMismatchedDigest(t *testing.T) {
	opts := workerOptions(t, "write")
	opts.ExpectedSHA256 = strings.Repeat("0", 64)
	if _, err := directexecution.Run(opts); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("expected digest mismatch: %v", err)
	}
}
func TestRunRejectsMissingCommand(t *testing.T) {
	opts := workerOptions(t, "write")
	opts.Command = "nonexistent-tei-direct-executable"
	if _, err := directexecution.Run(opts); err == nil {
		t.Fatal("expected command start error")
	}
}
