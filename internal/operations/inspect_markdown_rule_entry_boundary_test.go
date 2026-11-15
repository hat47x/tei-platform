package operations

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
)

// A selected Markdown rule is real read-only semantic Validation with a
// source byte anchor. It is not an implicitly published Function Entry.
// This test uses the product Workspace reader, not the Reference parser.
func TestSelectedRuleMarkdownSourceIsNotAnExecutableCapability(t *testing.T) {
	root := filepath.Join("..", "workspace", "testdata", "tei-md-rule")
	const ruleID = "sales.order_line.quantity_positive"
	subject := sourceindex.Subject{Kind: "semantic-validation", ID: ruleID}

	read, err := ReadWorkspaceSubject(root, subject)
	if err != nil {
		t.Fatalf("read selected Markdown rule: %v", err)
	}
	inspection := read.Inspection
	if read.ModelRevision == "" || inspection.Subject != subject ||
		inspection.Effect != "read-only" || inspection.SourceStatus != "indexed" ||
		inspection.SourceIndexFingerprint == "" {
		t.Fatalf("read did not bind rule to the product Source Index: %+v", inspection)
	}
	var canonical struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(inspection.Canonical, &canonical); err != nil {
		t.Fatalf("decode canonical rule: %v", err)
	}
	if canonical.ID != ruleID {
		t.Fatalf("wrong canonical rule: %q", canonical.ID)
	}
	if len(inspection.AuthorityLimits) == 0 {
		t.Fatal("source inspection must retain read-only authority limits")
	}

	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	artifactByPath := make(map[string]sourceindex.Artifact, len(inspection.Sources.Artifacts))
	for _, artifact := range inspection.Sources.Artifacts {
		artifactByPath[artifact.Path] = artifact
	}
	matched := false
	for _, ref := range inspection.Sources.References {
		if ref.Subject != subject || ref.Role != sourceindex.RoleDefinition {
			t.Fatalf("unexpected rule source reference: %+v", ref)
		}
		if ref.ArtifactPath != sourcePath {
			continue
		}
		artifact, ok := artifactByPath[ref.ArtifactPath]
		if !ok {
			t.Fatalf("missing artifact for rule reference: %+v", ref)
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
		if err != nil {
			t.Fatal(err)
		}
		if artifact.Fingerprint != sourceindex.Fingerprint(raw) {
			t.Fatal("rule Source Index fingerprint differs from actual Markdown bytes")
		}
		if err := ref.Range.Validate(raw); err != nil {
			t.Fatalf("invalid source byte range: %v", err)
		}
		if got := string(raw[ref.Range.Start.Offset:ref.Range.End.Offset]); got != ruleID {
			t.Fatalf("Markdown rule source anchor=%q, want %q", got, ruleID)
		}
		matched = true
	}
	if !matched {
		t.Fatalf("selected Markdown rule has no definition anchor in %s", sourcePath)
	}

	// A rule read as semantic.Validation never implies a published Function
	// or even the existence of a Canonical semantic.Capability with that ID.
	_, err = InspectSubject(root, sourceindex.Subject{Kind: "semantic-capability", ID: ruleID})
	var inspectionErr *InspectionError
	if !errors.As(err, &inspectionErr) || inspectionErr.Code != InspectionNotFound {
		t.Fatalf("Markdown rule became a Capability or unexpected error: %v", err)
	}
}
