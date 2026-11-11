package directexecution_test

import (
	"testing"

	"tei/internal/directexecution"
)

func TestRunRejectsMissingOutput(t *testing.T) {
	_, err := directexecution.Run(directexecution.Options{
		Command:    "true",
		OutputPath: "missing.json",
	})
	if err == nil {
		t.Fatal("expected error for missing output, got nil")
	}
}

func TestRunRecordsNonZeroExit(t *testing.T) {
	// On Windows, a failing binary may not exist; this is a known boundary.
	t.Skip("skipping non-zero exit test until a portable negative binary is provided")
}
