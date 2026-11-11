package directexecution_test

import (
	"os"
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
	if os.PathSeparator == '\\' {
		t.Skip("skipping non-zero exit test on Windows until portable negative binary is provided")
	}
	_, err := directexecution.Run(directexecution.Options{
		Command:    "false",
		OutputPath: t.TempDir() + "/output.json",
	})
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}
