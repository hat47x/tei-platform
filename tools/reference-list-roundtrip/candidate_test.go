package listroundtrip

import (
	"encoding/json"
	"strings"
	"testing"
)

var acceptRev = "sha256:" + strings.Repeat("c", 64)

func TestCandidateFromListEdit(t *testing.T) {
	// "no" -> "No" keeps the canonical segment "no".
	p, _ := Propose(world(), strings.NewReader(list(t, "attribute,s.order.no,s.order,no,No,REV")))
	raw, err := Candidate(p, acceptRev)
	if err != nil {
		t.Fatal(err)
	}
	var c candidate
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Delta) != 1 || c.Delta[0].Op != "set-symbol" || c.Delta[0].Kind != "attribute" ||
		c.Delta[0].ID != "s.order.no" || c.Delta[0].Symbol != "No" || c.BaseModelRevision != acceptRev {
		t.Fatalf("candidate = %s", raw)
	}
}

func TestCandidateRefusesIdentifierChangingRename(t *testing.T) {
	p, _ := Propose(world(), strings.NewReader(list(t, "attribute,s.order.no,s.order,no,orderNo,REV")))
	if _, err := Candidate(p, acceptRev); err == nil || !strings.Contains(err.Error(), "would change the identifier") {
		t.Fatalf("err = %v", err)
	}
}

func TestCandidateRefusesListWithDiagnosticsAndBadInput(t *testing.T) {
	// An ambiguous name leaves a diagnostic; nothing is guessed.
	amb, _ := Propose(world(), strings.NewReader(list(t, "attribute,,,no,No,")))
	if _, err := Candidate(amb, acceptRev); err == nil {
		t.Fatal("list with diagnostics accepted")
	}
	ok, _ := Propose(world(), strings.NewReader(list(t, "attribute,s.order.no,s.order,no,No,REV")))
	if _, err := Candidate(ok, "sha256:bad"); err == nil {
		t.Fatal("bad base revision accepted")
	}
	if _, err := Candidate(Proposal{BaseRevision: acceptRev}, acceptRev); err == nil {
		t.Fatal("empty list accepted")
	}
}
