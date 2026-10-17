package modelsearch

import (
	"encoding/json"
	"strings"
	"testing"
)

var rev = "sha256:" + strings.Repeat("b", 64)

func proposal(t *testing.T, text, with string, selected []string) Proposal {
	t.Helper()
	w := world()
	res, _ := Search(w, Query{Text: text, Exact: true})
	p, err := Propose(w, res, with, selected)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCandidateFromSymbolOnlyRename(t *testing.T) {
	// "orderNo" -> "OrderNo" keeps the canonical segment "order_no"? The fixture
	// IDs end in "no", so use a rename that keeps that segment.
	p := proposal(t, "orderNo", "No", []string{"sales.order.no"})
	raw, err := Candidate(p, rev)
	if err != nil {
		t.Fatal(err)
	}
	var c candidate
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Delta) != 1 || c.Delta[0].Op != "set-symbol" || c.Delta[0].Kind != "attribute" ||
		c.Delta[0].ID != "sales.order.no" || c.Delta[0].Symbol != "No" || c.BaseModelRevision != rev {
		t.Fatalf("candidate = %s", raw)
	}
}

func TestCandidateRefusesRenameThatChangesIdentifier(t *testing.T) {
	p := proposal(t, "orderNo", "salesNo", []string{"sales.order.no"})
	if _, err := Candidate(p, rev); err == nil || !strings.Contains(err.Error(), "would change the identifier") {
		t.Fatalf("err = %v", err)
	}
}

func TestCandidateRefusesDiagnosticsBadRevisionAndEmpty(t *testing.T) {
	collide := proposal(t, "orderNo", "customerNo", []string{"sales.order.no"})
	if _, err := Candidate(collide, rev); err == nil {
		t.Fatal("proposal with diagnostics accepted")
	}
	ok := proposal(t, "orderNo", "No", []string{"sales.order.no"})
	if _, err := Candidate(ok, "sha256:bad"); err == nil {
		t.Fatal("bad base revision accepted")
	}
	if _, err := Candidate(Proposal{BaseRevision: rev}, rev); err == nil {
		t.Fatal("empty proposal accepted")
	}
}
