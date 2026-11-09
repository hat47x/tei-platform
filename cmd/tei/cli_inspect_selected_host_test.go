package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestNormalCLIInspectUsesHostSelectedReferenceProvider(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	exit, err := inspectWorkspaceJSONSelected(&out, root, "", "semantic-data", "sales.order", 0)
	if err != nil || exit != 0 {
		t.Fatalf("selected normal CLI failed: exit=%d err=%v body=%s", exit, err, out.String())
	}
	var reply cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if !reply.OK || reply.Result == nil ||
		reply.Result.Subject.Kind != "semantic-data" ||
		reply.Result.Subject.ID != "sales.order" ||
		reply.Result.WorldFingerprint == "" {
		t.Fatalf("selected CLI projection incomplete: %+v", reply)
	}
}

func TestNormalCLIInspectDoesNotFallbackForMissingWorkspace(t *testing.T) {
	var out bytes.Buffer
	exit, err := inspectWorkspaceJSONSelected(&out, t.TempDir(), "", "semantic-data", "sales.order", 0)
	if err != nil || exit == 0 {
		t.Fatalf("unexpected success: exit=%d err=%v output=%s", exit, err, out.String())
	}
	var reply cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.OK || reply.Error == nil || reply.Error.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("missing Workspace did not fail at selected Provider Load: %+v", reply)
	}
}

func TestNormalCLIInspectInvalidArgsKeepJSONEnvelope(t *testing.T) {
	var out bytes.Buffer
	exit, err := inspectWorkspaceJSONSelected(&out, "", "", "", "", 0)
	if err != nil || exit != 2 {
		t.Fatalf("invalid CLI flags: exit=%d err=%v body=%s", exit, err, out.String())
	}
	var reply cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.OK || reply.Error == nil || reply.SchemaVersion != "tei.cli.inspect/v0" {
		t.Fatalf("CLI error envelope lost: %+v", reply)
	}
}
