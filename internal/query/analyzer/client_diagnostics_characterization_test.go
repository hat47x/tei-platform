package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
)

const syntheticMarker = "SYNTHETIC_SQL_TOKEN_PRIVATE_3719"

// This is a characterization of current behavior, NOT the desired policy.
// The test helper writes only artificial markers: no production SQL/secrets.
func TestWorkerDiagnosticBoundaryCharacterization(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_QX_DIAG_HELPER", "1")
	call := func(op string) (Result, error) {
		return RunPythonWorker(context.Background(), exe, "-test.run=^TestWorkerDiagnosticHelper$", Request{
			ProtocolVersion: 1, Operation: op, Query: core.CanonicalID("app.synthetic"),
			Source: "SELECT id FROM publicly_synthetic", // never real data
		})
	}
	t.Run("failure-stderr-appears-verbatim-in-Go-error", func(t *testing.T) {
		got, err := call("stderr-marker")
		if err == nil || !strings.Contains(err.Error(), syntheticMarker) {
			t.Fatalf("expected current stderr reflection: %v", err)
		}
		if got.Query != "" || got.Status != "" {
			t.Fatalf("result must be empty on failed process: %+v", got)
		}
	})
	t.Run("large-stderr-is-not-truncated-by-client", func(t *testing.T) {
		_, err := call("stderr-large")
		if err == nil {
			t.Fatal("expected error")
		}
		if got, want := strings.Count(err.Error(), "Z"), 131072; got != want {
			t.Fatalf("expected unbounded stderr capture %d Z chars, got %d", want, got)
		}
	})
	t.Run("success-diagnostic-is-forwarded-as-raw-result", func(t *testing.T) {
		got, err := call("diagnostic-marker")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got.Diagnostics) != 1 || got.Diagnostics[0].Message != syntheticMarker {
			t.Fatalf("diagnostic changed: %+v", got.Diagnostics)
		}
	})
	t.Run("success-empty-diagnostics", func(t *testing.T) {
		got, err := call("clean")
		if err != nil || got.Status != "ok" || len(got.Diagnostics) != 0 {
			t.Fatalf("result=%+v err=%v", got, err)
		}
	})
}

func TestWorkerDiagnosticHelper(t *testing.T) {
	if os.Getenv("TEI_QX_DIAG_HELPER") != "1" {
		return
	}
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		os.Exit(90)
	}
	switch req.Operation {
	case "stderr-marker":
		fmt.Fprint(os.Stderr, "worker diagnostic: "+syntheticMarker)
		os.Exit(3)
	case "stderr-large":
		fmt.Fprint(os.Stderr, strings.Repeat("Z", 131072))
		os.Exit(3)
	case "diagnostic-marker":
		_ = json.NewEncoder(os.Stdout).Encode(Result{ProtocolVersion: req.ProtocolVersion, Query: req.Query, Status: "error", Diagnostics: []Diagnostic{{Code: "SYNTHETIC", Message: syntheticMarker}}})
		os.Exit(0)
	default:
		_ = json.NewEncoder(os.Stdout).Encode(Result{ProtocolVersion: req.ProtocolVersion, Query: req.Query, Status: "ok"})
		os.Exit(0)
	}
}
