package verification

import (
	"context"
	"testing"
	"time"
)

func TestNewCLIRunReporter(t *testing.T) {
	reporter := NewCLIRunReporter("/tmp/workspace", "semantic-data", "sales.order", "tei", time.Minute)
	if reporter == nil {
		t.Fatal("expected non-nil reporter")
	}
	report, err := reporter.RunReport(context.Background(), true)
	if err != nil {
		t.Fatalf("RunReport returned error: %v", err)
	}
	if !report.OK {
		t.Fatalf("expected OK report, got %+v", report)
	}
}
