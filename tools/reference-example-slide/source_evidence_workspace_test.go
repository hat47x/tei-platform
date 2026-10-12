package exampleslide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func sourceAuditWorkspaceFixture(t *testing.T) (string, Input, AnswerInput) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	src := []byte("namespace sales\n\ndata Parent {\n    attribute id type std.identifier\n}\n\ndata Child {\n    attribute parentId type std.identifier\n    relation parent to sales.parent one\n}\n")
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), src, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := CatalogFromWorld(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	child, ok := catalog.Tables["sales.child"]
	if !ok || len(child.Fields) == 0 {
		t.Fatalf("fixture child not projected: %+v", catalog)
	}
	input := Input{
		Catalog: catalog,
		Example: Example{
			ID: "workspace-case", ModelRevision: catalog.Revision,
			Base: []Record{{ID: "child-example", Table: "sales.child",
				Values: map[string]json.RawMessage{child.Fields[0].ID: json.RawMessage(`"example"`)}}},
		},
		Diagram: Diagram{ID: "diagram", Placements: []Placement{
			{ID: "child", Table: "sales.child", X: 20, Y: 20},
			{ID: "parent", Table: "sales.parent", X: 360, Y: 20},
		}},
		Presentation: Presentation{Frames: []Frame{{ID: "base-slide", Stage: "base"}}},
	}
	review, err := Review(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Issues) != 1 || review.Issues[0].RelationID != "sales.child.parent" {
		t.Fatalf("expected one derived semantic relation issue: %+v", review.Issues)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Artifacts) != 1 {
		t.Fatalf("expected one source artifact: %+v", idx.Artifacts)
	}
	answer := AnswerInput{
		Schema: answerSchema, CaseID: review.CaseID,
		ModelRevision: review.ModelRevision, ReviewFingerprint: review.Fingerprint,
		Answers: []AnswerEntry{{
			Question:    QuestionRef{Kind: "derived", ID: review.Issues[0].ID},
			Response:    "関連のキー定義について確認が必要",
			Disposition: "investigate", Context: "受注例",
			ClaimedBy:   AnswerActor{Kind: "human", ID: "unverified"},
			Evidence: []EvidencePointer{{Kind: "source-index",
				Ref:         "semantic-relation:sales.child.parent",
				Fingerprint: idx.Artifacts[0].Fingerprint}},
		}},
	}
	return root, input, answer
}

func TestLoadedWorkspaceEvidenceUsesExactQualifiedRelationAnchor(t *testing.T) {
	root, input, answer := sourceAuditWorkspaceFixture(t)
	result, err := AuditLoadedWorkspaceAnswerSourceEvidence(root, input, answer)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 ||
		result.Findings[0].Status != "source-range-checked" ||
		result.Findings[0].ArtifactPath != "semantic/model.tei" {
		t.Fatalf("loaded source anchor did not match qualified subject: %+v", result)
	}
}

func TestLoadedWorkspaceEvidenceRejectsForgedCatalogAndStaleCase(t *testing.T) {
	root, input, answer := sourceAuditWorkspaceFixture(t)
	// A spoofed revision string is not enough: all projected meaning must
	// be exactly equal to the World loaded at this Workspace.
	input.Catalog.Relations[0].ID = "forged.relation"
	if _, err := AuditLoadedWorkspaceAnswerSourceEvidence(root, input, answer); err == nil ||
		!strings.Contains(err.Error(), "differs") {
		t.Fatalf("forged catalog was accepted: %v", err)
	}
	root, input, answer = sourceAuditWorkspaceFixture(t)
	src := filepath.Join(root, "semantic", "model.tei")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	// This is still syntactically valid, but it changes the canonical
	// semantic relation. The prior Case and answer must not be reused.
	changed := strings.Replace(string(data), "relation parent to", "relation vendor to", 1)
	if err := os.WriteFile(src, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := AuditLoadedWorkspaceAnswerSourceEvidence(root, input, answer); err == nil {
		t.Fatal("stale review survived changed Workspace semantics")
	}
}

func TestLoadedWorkspaceEvidenceCannotReadOutsideIndexedSnapshot(t *testing.T) {
	root, _, _ := sourceAuditWorkspaceFixture(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, reader, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.ReferencesFor(sourceindex.Subject{Kind: "semantic-relation", ID: "sales.child.parent"})) != 1 {
		t.Fatal("expected source relation anchor missing")
	}
	for _, path := range []string{"../secret.tei", "semantic/../secret.tei", "/etc/passwd", "semantic/missing.tei"} {
		if _, err := reader(path); err == nil {
			t.Fatalf("out-of-index source was available: %s", path)
		}
	}
}
