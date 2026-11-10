package sourceindex_test

import (
	"tei/internal/sourceindex"
	"testing"
)

func TestRevisionIdentifiesInventoryAndBytesRegardlessOfIndexOrder(t *testing.T) {
	a, _ := sourceindex.NewArtifact("semantic/a.tei", []byte("a"))
	b, _ := sourceindex.NewArtifact("applications/b.tei", []byte("b"))
	base := sourceindex.Revision(sourceindex.Index{Artifacts: []sourceindex.Artifact{a, b}})
	if sourceindex.Revision(sourceindex.Index{Artifacts: []sourceindex.Artifact{b, a}}) != base {
		t.Fatal("inventory order changed revision")
	}
	changed := b
	changed.Fingerprint = sourceindex.Fingerprint([]byte("new bytes"))
	if sourceindex.Revision(sourceindex.Index{Artifacts: []sourceindex.Artifact{a, changed}}) == base {
		t.Fatal("changed source bytes retained revision")
	}
	changed = b
	changed.Path = "applications/moved.tei"
	if sourceindex.Revision(sourceindex.Index{Artifacts: []sourceindex.Artifact{a, changed}}) == base {
		t.Fatal("moved source retained revision")
	}
	if sourceindex.Revision(sourceindex.Index{Artifacts: []sourceindex.Artifact{a}}) == base {
		t.Fatal("removed source retained revision")
	}
}
