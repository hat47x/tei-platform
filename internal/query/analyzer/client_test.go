package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"tei/internal/core"
)

func TestWorkerResponseMatchesRequestIdentity(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_SQL_ANALYZER_TEST_HELPER", "1")
	for _, tc := range []struct {
		name, operation string
		wantError       bool
	}{
		{"match", "match", false},
		{"matched-worker-error", "worker-error", false},
		{"other-query", "other-query", true},
		{"missing-query", "missing-query", true},
		{"other-version", "other-version", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := Request{ProtocolVersion: 1, Operation: tc.operation, Query: core.MustCanonicalID("app.current")}
			got, err := RunPythonWorker(context.Background(), exe, "-test.run=^TestSQLAnalyzerWorkerHelper$", req)
			if (err != nil) != tc.wantError {
				t.Fatalf("result=%+v err=%v wantError=%v", got, err, tc.wantError)
			}
			if tc.wantError && !reflect.DeepEqual(got, Result{}) {
				t.Fatalf("mismatched result must not escape: %+v", got)
			}
			if tc.operation == "worker-error" && (got.Status != "error" || len(got.Diagnostics) != 1) {
				t.Fatalf("matched worker diagnostics were lost: %+v", got)
			}
		})
	}
	for _, req := range []Request{
		{ProtocolVersion: 1},
		{Query: core.MustCanonicalID("app.current")},
		{ProtocolVersion: -1, Query: core.MustCanonicalID("app.current")},
	} {
		if _, err := RunPythonWorker(context.Background(), exe, "-test.run=^TestSQLAnalyzerWorkerHelper$", req); err == nil {
			t.Fatalf("incomplete request was accepted: %+v", req)
		}
	}
}

func TestSQLAnalyzerWorkerHelper(t *testing.T) {
	if os.Getenv("TEI_SQL_ANALYZER_TEST_HELPER") != "1" {
		return
	}
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		os.Exit(2)
	}
	result := Result{ProtocolVersion: req.ProtocolVersion, Status: "ok", Query: req.Query}
	switch req.Operation {
	case "worker-error":
		result.Status = "error"
		result.Diagnostics = []Diagnostic{{Code: "TEST", Message: "parse failure"}}
	case "other-query":
		result.Query = core.MustCanonicalID("app.other")
	case "missing-query":
		result.Query = ""
	case "other-version":
		result.ProtocolVersion++
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}
