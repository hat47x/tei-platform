package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestOperationWriteIntentRemainsOptInForWorkspaceCLI(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	args := []string{
		"-workspace", root,
		"-application", "order_entry",
		"-subject-kind", "application-pattern",
		"-subject-id", "order_entry.submit_order",
	}
	var buf bytes.Buffer
	if err := runCLI(args, &buf); err != nil {
		t.Fatal(err)
	}
	var legacy map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &legacy); err != nil {
		t.Fatal(err)
	}
	if _, exists := legacy["operationWriteIntent"]; exists {
		t.Fatalf("opt-in operation context leaked into legacy JSON: %s", buf.String())
	}
	if _, exists := legacy["assessment"]; exists {
		t.Fatal("legacy output was unexpectedly wrapped")
	}

	buf.Reset()
	withPurpose := append(append([]string(nil), args...), "-purpose", "execute-preflight")
	if err := runCLI(withPurpose, &buf); err != nil {
		t.Fatal(err)
	}
	var projected struct {
		Assessment WorkspaceAssessment `json:"assessment"`
		Readiness  PurposeProjection  `json:"readiness"`
	}
	if err := json.Unmarshal(buf.Bytes(), &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Assessment.OperationWriteIntent == nil ||
		projected.Assessment.OperationWriteIntent.Status != "effect-not-declared" ||
		projected.Readiness.ExecutionAllowed || projected.Assessment.ExecutionAllowed {
		t.Fatalf("missing bounded opt-in write intent or runtime authority leaked: %+v", projected)
	}
}
