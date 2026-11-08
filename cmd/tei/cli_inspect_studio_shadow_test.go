package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/studiohost"
)

// This deliberately compares TWO existing direct-operation consumers. It pins
// D1a shadow behavior before the tooling-scope Inspector Provider is wired.
// Passing does not show that a Plugin was selected, built, loaded, or invoked.
func TestDirectInspectorCLIStudioShadowParity(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	for _, subject := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-validation", ID: "sales.order_line.line_number_positive"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
		// #3482 independently fixes the public Capability/Application slice.
		// This adds a *second consumer* baseline, not a Function/Provider proof.
		{Kind: "semantic-capability", ID: "sales.order.submit"},
		{Kind: "application", ID: "order_entry"},
		{Kind: "application-pattern", ID: "order_entry.submit_order"},
	} {
		t.Run(subject.Kind+"/"+subject.ID, func(t *testing.T) {
			var out bytes.Buffer
			code, err := inspectWorkspaceJSON(&out, root, "", subject.Kind, subject.ID, 0)
			if err != nil || code != 0 {
				t.Fatalf("CLI inspect code=%d err=%v out=%s", code, err, out.String())
			}
			var cli cliInspectionReport
			if err := json.Unmarshal(out.Bytes(), &cli); err != nil {
				t.Fatalf("CLI response is invalid JSON: %v", err)
			}
			studio := host.InspectSubject("workspace-1", subject.Kind, subject.ID)
			if !cli.OK || cli.Result == nil || cli.Error != nil ||
				!studio.OK || studio.Value == nil || studio.Failure != nil {
				t.Fatalf("consumer results disagree: CLI=%+v Studio=%+v", cli, studio)
			}
			if cli.SchemaVersion != "tei.cli.inspect/v0" ||
				cli.Result.SchemaVersion != "tei.operations.inspect/v0" ||
				cli.Result.Effect != "read-only" || cli.Result.Subject != subject {
				t.Fatalf("CLI projection has unexpected identity or effect: %+v", cli)
			}
			if !reflect.DeepEqual(*cli.Result, *studio.Value) {
				t.Fatalf("different direct-operation projections: CLI=%+v Studio=%+v", *cli.Result, *studio.Value)
			}
			if cli.Result.Workspace == nil ||
				cli.Result.WorldFingerprint == "" ||
				cli.Result.SourceIndexFingerprint == "" ||
				len(cli.Result.Canonical) == 0 {
				t.Fatalf("missing baseline identity or Source observation: %+v", *cli.Result)
			}
		})
	}
}

// Different transport envelopes may still preserve the exact failure reason.
// The Studio allowlist is a separate host boundary and is not supplied by the
// CLI; this compares only a request to an allowed Workspace.
func TestDirectInspectorCLIStudioMissingSubjectParity(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	host, err := studiohost.NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	const missing = "sales.absent_inspector_d1_subject"
	var out bytes.Buffer
	code, err := inspectWorkspaceJSON(&out, root, "", "semantic-data", missing, 0)
	if err != nil || code != 1 {
		t.Fatalf("CLI missing Subject code=%d err=%v out=%s", code, err, out.String())
	}
	var cli cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &cli); err != nil {
		t.Fatal(err)
	}
	studio := host.InspectSubject("workspace-1", "semantic-data", missing)
	if cli.OK || cli.Result != nil || cli.Error == nil ||
		cli.Error.Code != "SUBJECT_NOT_FOUND" ||
		studio.OK || studio.Value != nil || studio.Failure == nil ||
		studio.Failure.Code != cli.Error.Code {
		t.Fatalf("missing Subject was silently reclassified: CLI=%+v Studio=%+v", cli, studio)
	}
}
