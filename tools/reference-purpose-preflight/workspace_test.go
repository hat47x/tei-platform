package main

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWorkspaceSubjectHasVerifiedCurrentSourceButNoBusinessRuleAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	first, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.customer_reference")
	if err != nil {
		t.Fatal(err)
	}
	if first.SourceStatus != "traceable-source-declaration" {
		t.Fatalf("real Source Index lookup was not qualified: %+v", first)
	}
	if first.RuleMeaning != "not-established" || first.ExecutionAllowed {
		t.Fatalf("source declaration was wrongly promoted to business meaning/authority: %+v", first)
	}
	if len(first.Witnesses) == 0 {
		t.Fatalf("missing actual source evidence: %+v", first)
	}
	for _, witness := range first.Witnesses {
		if witness.Path == "" || witness.Fingerprint == "" {
			t.Fatalf("incomplete Source Index evidence: %+v", witness)
		}
	}
	second, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.customer_reference")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("source witness changed for unchanged source: first=%+v second=%+v", first, second)
	}
}

func TestWorkspaceSourceDoesNotInventMissingPattern(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	got, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.nonexistent_action")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceStatus != "unresolved" || got.RuleMeaning != "not-established" || got.ExecutionAllowed || len(got.Witnesses) != 0 {
		t.Fatalf("missing Pattern was treated as resolved: %+v", got)
	}
	if len(got.Unresolved) == 0 || got.Unresolved[0] != "application-pattern-not-declared" {
		t.Fatalf("expected explicit diagnosis: %+v", got)
	}
}

func TestWorkspaceModeRejectsUnknownScopeAndMixedInputs(t *testing.T) {
	for _, args := range [][]string{
		{"-workspace", "../../reference/workspace", "-application", "order_entry", "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
		{"-input", "fixture.json", "-workspace", "../../reference/workspace", "-application", "order_entry", "-subject-kind", "application-pattern", "-subject-id", "order_entry.customer_reference"},
		{"-workspace", "../../reference/workspace", "-subject-kind", "application-pattern"},
	} {
		var output bytes.Buffer
		if err := runCLI(args, &output); err == nil {
			t.Fatalf("invalid arguments passed: %v", args)
		}
		if strings.Contains(output.String(), "executionAllowed") {
			t.Fatalf("invalid input emitted success-like report: %v", args)
		}
	}
}
