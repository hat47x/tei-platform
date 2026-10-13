package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestCLIPurposeProjectionHasBoundedObligationsAndNoAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	base := []string{"-workspace", root, "-application", "order_entry", "-subject-kind", "semantic-validation", "-subject-id", "sales.order_line.line_number_positive"}
	for _, tc := range []struct {
		purpose, expectedRequirement string
	}{
		{"describe", "typed-source"},
		{"simulate", "evaluation-fixture"},
		{"execute-preflight", "effect-fence"},
	} {
		t.Run(tc.purpose, func(t *testing.T) {
			var output bytes.Buffer
			if err := runCLI(append(append([]string{}, base...), "-purpose", tc.purpose), &output); err != nil {
				t.Fatal(err)
			}
			var result struct {
				Assessment WorkspaceAssessment `json:"assessment"`
				Readiness  PurposeProjection   `json:"readiness"`
			}
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Readiness.Purpose != tc.purpose || result.Assessment.ExecutionAllowed || result.Readiness.ExecutionAllowed {
				t.Fatalf("incorrect purpose/authority: %+v", result)
			}
			if _, ok := purposeRequirementByName(result.Readiness, tc.expectedRequirement); !ok {
				t.Fatalf("missing %s in %+v", tc.expectedRequirement, result.Readiness)
			}
			if result.Assessment.RuleMeaning != "canonical-validation-declared" {
				t.Fatalf("Validation declaration unavailable: %+v", result.Assessment)
			}
		})
	}
}

func TestCLIRejectsPurposeInFixtureModeAndUnknownPurpose(t *testing.T) {
	for _, args := range [][]string{
		{"-input", "order-approval.json", "-purpose", "describe"},
		{"-workspace", "../../reference/workspace", "-application", "order_entry", "-subject-kind", "semantic-validation", "-subject-id", "sales.order_line.line_number_positive", "-purpose", "execute"},
	} {
		var output bytes.Buffer
		if err := runCLI(args, &output); err == nil {
			t.Fatalf("invalid combination was accepted: %v", args)
		}
		if output.Len() != 0 {
			t.Fatalf("invalid request emitted success-shaped output: %s", output.String())
		}
	}
}
