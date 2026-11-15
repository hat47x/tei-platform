package operations

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// A selected Markdown Rule becomes semantic.Validation, not a published
// Function Entry. Resolve the positive Rule and the missing Capability from
// exactly one product Workspace Load and its Source Index.
func TestSelectedRuleMarkdownSourceIsNotAnExecutableCapability(t *testing.T) {
	root := filepath.Join("..", "workspace", "testdata", "tei-md-rule")
	const ruleID = "sales.order_line.quantity_positive"
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load product Markdown Rule Workspace: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
		t.Fatalf("unexpected selected product Rule language: %+v", loaded.AuthoringSources)
	}
	idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("index same-Load Source: %v", err)
	}

	subject := sourceindex.Subject{Kind: "semantic-validation", ID: ruleID}
	inspection, err := InspectionFromLoad(loaded, idx, subject)
	if err != nil {
		t.Fatalf("inspect selected Rule: %v", err)
	}
	var canonical struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(inspection.Canonical, &canonical); err != nil {
		t.Fatalf("decode canonical Rule: %v", err)
	}
	if canonical.ID != ruleID || inspection.Effect != "read-only" ||
		inspection.SourceStatus != "indexed" || len(inspection.AuthorityLimits) == 0 {
		t.Fatalf("Rule Source observation lost its identity or limits: %+v", inspection)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ref := range inspection.Sources.References {
		if ref.Subject != subject || ref.Role != sourceindex.RoleDefinition {
			t.Fatalf("Rule Reference changed identity or role: %+v", ref)
		}
		if ref.ArtifactPath != sourcePath {
			continue
		}
		if err := ref.Range.Validate(raw); err != nil {
			t.Fatalf("invalid Rule byte range: %v", err)
		}
		if got := string(raw[ref.Range.Start.Offset:ref.Range.End.Offset]); got != ruleID {
			t.Fatalf("Rule source anchor=%q, want %q", got, ruleID)
		}
		found = true
	}
	if !found {
		t.Fatalf("Rule %s has no original Markdown definition Source", ruleID)
	}

	// A Source lookup can be empty without proving whether a Canonical model
	// exists. The same-Load Canonical Inspector must reject a Capability with
	// the Rule's ID; a readable Rule cannot become a callable Function.
	capability := sourceindex.Subject{Kind: "semantic-capability", ID: ruleID}
	trace, err := TraceSubjectSourcesFromIndex(idx, capability)
	if err != nil || len(trace.References) != 0 {
		t.Fatalf("Rule ID unexpectedly has Capability Source: trace=%+v err=%v", trace, err)
	}
	_, err = InspectionFromLoad(loaded, idx, capability)
	var inspectionErr *InspectionError
	if !errors.As(err, &inspectionErr) || inspectionErr.Code != InspectionNotFound {
		t.Fatalf("Rule was mistaken for a Capability: %v", err)
	}
}
