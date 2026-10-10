package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/modelref"
	"tei/internal/operations"
	"tei/internal/subject"
)

func parseTraceReport(t *testing.T, payload []byte) cliTraceReport {
	t.Helper()
	var result cliTraceReport
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("trace did not return a single JSON document: %v: %s", err, payload)
	}
	if result.SchemaVersion != "tei.cli.trace/v0" {
		t.Fatalf("wrong CLI trace schema: %+v", result)
	}
	return result
}

func TestTraceWorkspaceJSONSuccessAndContinuation(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	run := func(code int, direction, mode, fingerprint string, offset, limit int) cliTraceReport {
		t.Helper()
		var out bytes.Buffer
		got, err := traceWorkspaceJSON(&out, root, "",
			"reference-observation", "semantic-relation", "sales.order.customer",
			direction, mode, fingerprint, offset, limit, 0)
		if err != nil || got != code {
			t.Fatalf("exit=%d error=%v body=%s", got, err, out.String())
		}
		return parseTraceReport(t, out.Bytes())
	}
	summary := run(0, "both", "summary", "", 0, 0)
	if !summary.OK || summary.Result == nil ||
		summary.Result.Trace.Total < 2 || len(summary.Result.Trace.Rows) != 0 {
		t.Fatalf("bad trace summary: %+v", summary)
	}
	page := run(0, "both", "full", summary.Result.SnapshotFingerprint, 0, 1)
	if !page.OK || page.Result == nil || len(page.Result.Trace.Rows) != 1 ||
		page.Result.Trace.NextOffset == nil {
		t.Fatalf("bad trace page: %+v", page)
	}
	next := run(0, "both", "full", page.Result.SnapshotFingerprint,
		*page.Result.Trace.NextOffset, 1)
	if !next.OK || next.Result == nil || len(next.Result.Trace.Rows) == 0 {
		t.Fatalf("bad trace continuation: %+v", next)
	}
	// CLI JSON is only a transport projection. For the same Workspace and
	// selector it must agree with the typed, in-process Go operation.
	fromGo, err := operations.TraceWorkspace(root, modelref.Key{
		Scope: "reference-observation",
		Subject: subject.Subject{Kind: "semantic-relation", ID: "sales.order.customer"},
	}, modelref.TraceOptions{
		Direction: "both", Mode: "full",
		Limit: 1, ExpectedFingerprint: summary.Result.SnapshotFingerprint,
	})
	if err != nil {
		t.Fatalf("typed operation trace failed: %v", err)
	}
	if !reflect.DeepEqual(*page.Result, fromGo) {
		t.Fatalf("CLI/Go semantic results differ\nCLI: %+v\nGo: %+v", *page.Result, fromGo)
	}
	changedDirection := run(1, "incoming", "full",
		summary.Result.SnapshotFingerprint, 1, 1)
	if changedDirection.OK || changedDirection.Error == nil ||
		changedDirection.Error.Code != "STALE_SNAPSHOT" {
		t.Fatalf("trace continuation changed direction without detection: %+v", changedDirection)
	}
	stale := run(1, "both", "full", "sha256:obsolete", 1, 1)
	if stale.OK || stale.Error == nil || stale.Error.Code != "STALE_SNAPSHOT" {
		t.Fatalf("stale continuation not rejected: %+v", stale)
	}
}

func TestTraceWorkspaceJSONArgumentErrors(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	cases := []struct {
		root, world, scope, kind, id, direction, mode string
		offset, limit, positional, expectedExit       int
		code                                          string
	}{
		{root: root, scope: "", kind: "semantic-data", id: "sales.order", expectedExit: 2, code: "INVALID_ARGUMENT"},
		{root: root, world: "world.json", scope: "test", kind: "semantic-data", id: "sales.order", expectedExit: 2, code: "INVALID_ARGUMENT"},
		{root: root, scope: "test", kind: "semantic-data", id: "sales.order", direction: "sideways", expectedExit: 2, code: "INVALID_ARGUMENT"},
		{root: root, scope: "test", kind: "semantic-data", id: "missing", expectedExit: 1, code: "SUBJECT_NOT_FOUND"},
		{root: root, scope: "test", kind: "semantic-data", id: "sales.order", mode: "summary", limit: 1, expectedExit: 2, code: "INVALID_ARGUMENT"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		exit, err := traceWorkspaceJSON(&out, tc.root, tc.world, tc.scope, tc.kind, tc.id,
			tc.direction, tc.mode, "", tc.offset, tc.limit, tc.positional)
		if err != nil || exit != tc.expectedExit {
			t.Fatalf("error case %+v: exit=%d err=%v stdout=%q", tc, exit, err, out.String())
		}
		report := parseTraceReport(t, out.Bytes())
		if report.OK || report.Error == nil || report.Error.Code != tc.code {
			t.Fatalf("wrong error code case %+v: %+v", tc, report)
		}
	}
}
