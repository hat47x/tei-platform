package main

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/workspace"
)

// A Canonical Contract Query declaration is evidence of an intended query,
// not evidence that a concrete Provider is bound, authorized, or operational.
// It also does not prove the absence of other side effects at a Target.
func TestReferenceContractQueryDoesNotProveProviderOrEffectAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}

	matches := 0
	for _, app := range loaded.World.Applications {
		if string(app.ID) != "order_entry" {
			continue
		}
		for _, pattern := range app.Patterns {
			if string(pattern.ID) != "order_entry.customer_reference" {
				continue
			}
			matches++
			if pattern.Kind != application.PatternReference || pattern.ContractQuery == nil {
				t.Fatalf("expected real, declared reference Contract Query: %+v", pattern)
			}
			if string(pattern.ContractQuery.Contract) != "master_data.customer" ||
				string(pattern.ContractQuery.Query) != "find_by_id" {
				t.Fatalf("unexpected real Contract Query declaration: %+v", pattern.ContractQuery)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("real Reference Pattern not uniquely found: count=%d", matches)
	}

	assessment, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.customer_reference")
	if err != nil {
		t.Fatal(err)
	}
	if assessment.SourceStatus != "traceable-source-declaration" ||
		assessment.OperationWriteIntent == nil ||
		assessment.OperationWriteIntent.Status != "not-command" {
		t.Fatalf("reference Contract Query conflated with a write effect: %+v", assessment)
	}

	readiness, err := ProjectForPurpose(assessment, "execute-preflight")
	if err != nil {
		t.Fatal(err)
	}
	provider, ok := purposeRequirementByName(readiness, "provider-contract")
	if !ok || provider.Status != "conditional" {
		t.Fatalf("Contract Query wrongly certified Provider binding: %+v", readiness)
	}
	fence, ok := purposeRequirementByName(readiness, "effect-fence")
	if !ok || fence.Status != "conditional" {
		t.Fatalf("non-command classification was mistaken for effect safety proof: %+v", readiness)
	}
	if readiness.ExecutionAllowed || assessment.ExecutionAllowed {
		t.Fatal("static query declaration granted runtime execution")
	}
}
