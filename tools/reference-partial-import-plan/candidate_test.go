package partialimport

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCandidateCarriesOnlyAdditions(t *testing.T) {
	rev := "sha256:" + strings.Repeat("a", 64)
	src := w(typ, order("t.code"))
	plan, _ := Analyze(src, w(typ), []string{"s.order"})
	raw, err := Candidate(plan, src, rev)
	if err != nil {
		t.Fatal(err)
	}
	var c candidate
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.BaseModelRevision != rev || len(c.Delta) != 1 || c.Delta[0].Op != "add-data" ||
		c.Delta[0].Data.ID != "s.order" || c.Delta[0].Placement != "semantic/s/order.tei" {
		t.Fatalf("candidate = %s", raw)
	}
	if c.Delta[0].Data.Attributes[0].Type != "t.code" {
		t.Fatalf("attribute type lost: %s", raw)
	}
}

func TestCandidateRefusesUpdatesConflictsAndBadRevision(t *testing.T) {
	rev := "sha256:" + strings.Repeat("a", 64)
	src := w(typ, order("t.code"))
	same, _ := Analyze(src, w(typ, order("t.code")), []string{"s.order"})
	if _, err := Candidate(same, src, rev); err == nil {
		t.Fatal("identical item accepted")
	}
	missing, _ := Analyze(src, w(nil), []string{"s.order"})
	if _, err := Candidate(missing, src, rev); err == nil {
		t.Fatal("unresolved dependency accepted")
	}
	ok, _ := Analyze(src, w(typ), []string{"s.order"})
	if _, err := Candidate(ok, src, "sha256:bad"); err == nil {
		t.Fatal("bad base revision accepted")
	}
}
