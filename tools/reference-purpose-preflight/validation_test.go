package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalValidationSourceStillRequiresIndependentAcceptance(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	got, err := InspectValidation(root, "order_entry", "sales.order_line.line_number_positive")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceStatus != "traceable-source-declaration" || got.RuleMeaning != "canonical-validation-declared" {
		t.Fatalf("real validation declaration not traced: %+v", got)
	}
	if got.ExecutionAllowed {
		t.Fatalf("a source declaration granted execution: %+v", got)
	}
	if len(got.Witnesses) == 0 {
		t.Fatalf("no current Source Index witness: %+v", got)
	}
	foundSource := false
	for _, witness := range got.Witnesses {
		if witness.Path == "semantic/sales/validation.tei" && strings.HasPrefix(witness.Fingerprint, "sha256:") {
			foundSource = true
		}
	}
	if !foundSource {
		t.Fatalf("missing typed Validation source witness: %+v", got.Witnesses)
	}
	foundHuman, foundRuntime := false, false
	for _, code := range got.Unresolved {
		if code == "human-acceptance-not-verified" {
			foundHuman = true
		}
		if code == "runtime-validation-not-evaluated" {
			foundRuntime = true
		}
	}
	if !foundHuman || !foundRuntime {
		t.Fatalf("authorship was wrongly promoted to acceptance or execution: %+v", got)
	}
}

func TestValidationMustBeDeclaredAndExplicitlyUsedByApplication(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, tc := range []struct {
		name, application, id, diagnostic string
	}{
		{"unknown", "order_entry", "sales.order_line.not_declared", "semantic-validation-not-declared"},
		{"not-used-by-application", "repair_intake", "sales.order_line.line_number_positive", "application-validation-use-not-established"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InspectValidation(root, tc.application, tc.id)
			if err != nil {
				t.Fatal(err)
			}
			if got.ExecutionAllowed || got.SourceStatus != "unresolved" || got.RuleMeaning != "not-established" {
				t.Fatalf("invalid validation considered qualified: %+v", got)
			}
			found := false
			for _, code := range got.Unresolved {
				found = found || code == tc.diagnostic
			}
			if !found {
				t.Fatalf("missing diagnostic %s in %+v", tc.diagnostic, got)
			}
		})
	}
}

func TestValidationCLIReadOnlyReport(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	args := []string{"-workspace", root, "-application", "order_entry", "-subject-kind", "semantic-validation", "-subject-id", "sales.order_line.line_number_positive"}
	var buf bytes.Buffer
	if err := runCLI(args, &buf); err != nil {
		t.Fatal(err)
	}
	var report WorkspaceAssessment
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.RuleMeaning != "canonical-validation-declared" || report.ExecutionAllowed || report.SourceStatus != "traceable-source-declaration" {
		t.Fatalf("validation CLI report falsely claimed execution or lacked source: %+v", report)
	}
}
