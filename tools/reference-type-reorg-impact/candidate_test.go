package typereorg

import (
	"encoding/json"
	"strings"
	"testing"
)

var acceptRev = "sha256:" + strings.Repeat("d", 64)

func TestCandidateFromSameRepresentationMerge(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.client_code"})
	raw, err := Candidate(imp, acceptRev)
	if err != nil {
		t.Fatal(err)
	}
	var c candidate
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Delta) != 1 || c.Delta[0].Op != "set-attribute-type" || c.Delta[0].ID != "s.order.cust" ||
		c.Delta[0].AttributeType != "s.client_code" || c.BaseModelRevision != acceptRev {
		t.Fatalf("candidate = %s", raw)
	}
}

func TestCandidateRefusesMergesThatNeedConversionOrAreMalformed(t *testing.T) {
	// s.client_code is based on std.string; moving it into s.item_code keeps the
	// representation, but its alias s.short_cust would be left dangling.
	alias, _ := Analyze(world(), Merge{From: []string{"s.client_code"}, Into: "s.item_code"})
	if _, err := Candidate(alias, acceptRev); err == nil {
		t.Fatal("merge leaving an alias type accepted")
	}
	storage, _ := Analyze(world(), Merge{From: []string{"s.cust_code", "s.item_code"}, Into: "s.client_code"})
	if _, err := Candidate(storage, acceptRev); err == nil {
		t.Fatal("merge with different physical storage accepted")
	}
	self, _ := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.cust_code"})
	if _, err := Candidate(self, acceptRev); err == nil {
		t.Fatal("merge into itself accepted")
	}
	prim, _ := Analyze(world(), Merge{From: []string{"std.string"}, Into: "s.cust_code"})
	if _, err := Candidate(prim, acceptRev); err == nil {
		t.Fatal("primitive source accepted")
	}
}

func TestCandidateRefusesBadRevisionAndUnusedMerge(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.client_code"})
	if _, err := Candidate(imp, "sha256:bad"); err == nil {
		t.Fatal("bad base revision accepted")
	}
	unused, _ := Analyze(world(), Merge{From: []string{"s.short_cust"}, Into: "s.client_code"})
	if _, err := Candidate(unused, acceptRev); err == nil {
		t.Fatal("merge with no attribute use accepted")
	}
}
// A split impact reuses the merge-shaped display payload. Even with an
// explicit reassignment, the merge-only Candidate API must refuse it;
// otherwise it would silently write all attributes to the first target.
func TestCandidateRefusesSplitImpact(t *testing.T) {
	w := splitWorld()
	imp, err := SplitImpact(w, Split{
		From: "sales.order_id",
		Into: []string{"sales.order_status"},
		Reassignment: map[string]string{
			"sales.order.id": "sales.order_status",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !codes(imp.Diagnostics)["split-candidate-unsupported"] {
		t.Fatalf("missing split candidate guard diagnostic: %+v", imp.Diagnostics)
	}
	if _, err := Candidate(imp, acceptRev); err == nil ||
		!strings.Contains(err.Error(), "split-candidate-unsupported") {
		t.Fatalf("Candidate accepted split Impact or returned wrong error: %v", err)
	}
}
