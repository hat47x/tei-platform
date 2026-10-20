package generatorselection

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoCandidateRecordsBuildEvidence(t *testing.T) {
	workspaceDir, contract, fixtures, candidates := paths(t)
	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, candidate := range receipt.Candidates {
		if candidate.Language != LanguageGo {
			continue
		}
		checked++
		evidence := candidate.BuildEvidence
		if evidence == nil {
			t.Fatalf("%s has no build evidence", candidate.ID)
		}
		if evidence.Language != LanguageGo || evidence.Command == "" || evidence.Toolchain == "" {
			t.Fatalf("%s has incomplete build evidence: %+v", candidate.ID, evidence)
		}
		if evidence.GOOS == "" || evidence.GOARCH == "" {
			t.Fatalf("%s has no GOOS/GOARCH: %+v", candidate.ID, evidence)
		}
		if evidence.SourceDigest != candidate.SourceDigest {
			t.Fatalf("%s source digest does not match build evidence", candidate.ID)
		}
		if evidence.BinaryDigest != candidate.BinaryDigest {
			t.Fatalf("%s binary digest does not match build evidence", candidate.ID)
		}
		if evidence.Reproducibility != "not-tested" {
			t.Fatalf("%s made a reproducibility claim: %+v", candidate.ID, evidence)
		}
		source, err := os.ReadFile(filepath.Join(candidates, candidate.ID, "main.go"))
		if err != nil {
			t.Fatal(err)
		}
		want := buildInputDigest(source, evidence.Toolchain, evidence.GOOS, evidence.GOARCH, evidence.CGOEnabled, evidence.GOFLAGS)
		if evidence.BuildInputDigest != want {
			t.Fatalf("%s build input digest mismatch: %q want %q", candidate.ID, evidence.BuildInputDigest, want)
		}
	}
	if checked != len(receipt.Candidates) {
		t.Fatalf("expected %d Go candidates, checked %d", len(receipt.Candidates), checked)
	}
}
