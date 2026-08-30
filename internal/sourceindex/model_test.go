package sourceindex_test

import (
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func TestRangeUsesByteOffsets(t *testing.T) {
	content := []byte("a\nあb\n")

	r, err := sourceindex.RangeFromOffsets(content, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if r.Start.Offset != 2 || r.Start.Line != 2 || r.Start.Column != 1 {
		t.Fatalf("unexpected start: %+v", r.Start)
	}
	if r.End.Offset != 5 || r.End.Line != 2 || r.End.Column != 4 {
		t.Fatalf("unexpected end: %+v", r.End)
	}
	if err := r.Validate(content); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactFingerprintDetectsStaleContent(t *testing.T) {
	content := []byte("query sales.recent_orders {}\n")
	artifact, err := sourceindex.NewArtifact("queries/recent_orders.tei", content)
	if err != nil {
		t.Fatal(err)
	}
	idx := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact}}

	if err := idx.Validate(func(string) ([]byte, error) { return content, nil }); err != nil {
		t.Fatalf("fresh index rejected: %v", err)
	}
	changed := []byte("query sales.recent_orders { datasource sales }\n")
	if err := idx.Validate(func(string) ([]byte, error) { return changed, nil }); err == nil || !strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("stale fingerprint was not rejected: %v", err)
	}
}

func TestWorkspacePathNormalizationRejectsEscape(t *testing.T) {
	for _, raw := range []string{"../secret.sql", "/tmp/query.sql", `C:\\temp\\query.sql`} {
		if _, err := sourceindex.NormalizeWorkspacePath(raw); err == nil {
			t.Fatalf("unsafe path %q was accepted", raw)
		}
	}
	got, err := sourceindex.NormalizeWorkspacePath(`applications\\order-entry\\query.sql`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "applications/order-entry/query.sql" {
		t.Fatalf("normalized path=%q", got)
	}
}

func TestIndexReferencesForSubject(t *testing.T) {
	content := []byte("application order_entry {}\n")
	artifact, err := sourceindex.NewArtifact("applications/order-entry/application.tei", content)
	if err != nil {
		t.Fatal(err)
	}
	r, err := sourceindex.RangeFromOffsets(content, 12, 23)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "application", ID: "order_entry"}
	idx := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{artifact},
		References: []sourceindex.Reference{{
			Subject:      subject,
			ArtifactPath: artifact.Path,
			Range:        r,
			Role:         sourceindex.RoleDefinition,
		}},
	}
	idx.Normalize()

	refs := idx.ReferencesFor(subject)
	if len(refs) != 1 || refs[0].ArtifactPath != artifact.Path {
		t.Fatalf("references=%#v", refs)
	}
	if err := idx.Validate(func(string) ([]byte, error) { return content, nil }); err != nil {
		t.Fatal(err)
	}
}
