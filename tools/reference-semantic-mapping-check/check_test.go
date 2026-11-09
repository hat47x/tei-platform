package semanticmappingcheck

import (
	"strings"
	"testing"
)

func acc(def string) Mapping {
	return Mapping{Definition: def, Status: MappingAccepted, Origin: OriginHuman, AcceptedBy: "owner"}
}

func base() Input {
	return Input{
		Schema:      InputSchema,
		Definitions: []Definition{{ID: "d1"}, {ID: "d2"}},
		Elements: []Element{
			{ID: "a", ObservedIn: "t.a", Mappings: []Mapping{acc("d1")}},
			{ID: "b", Mappings: []Mapping{acc("d1")}},
			{ID: "c"},
			{ID: "d"},
		},
	}
}

func TestFixture(t *testing.T) {
	in, err := LoadInput("testdata/input.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Check(in)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusRejected || r.Defined != 3 {
		t.Fatalf("status=%s defined=%d", r.Status, r.Defined)
	}
	if strings.Join(r.Ambiguous, ",") != "e04" {
		t.Fatalf("ambiguous %v", r.Ambiguous)
	}
	if strings.Join(r.Unfilled, ",") != "e05,e06,e07" {
		t.Fatalf("unfilled %v", r.Unfilled)
	}
}

func TestAICandidateStaysUnfilled(t *testing.T) {
	in := base()
	in.Elements[2].Mappings = []Mapping{{Definition: "d2", Status: MappingCandidate, Origin: OriginAI}}
	r, err := Check(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range r.Elements {
		if e.ElementID == "c" && (e.State != StateUnfilled || len(e.Candidates) != 1 || len(e.Why) != 0) {
			t.Fatalf("%+v", e)
		}
	}
}

func TestWhatAndWhyAreSeparate(t *testing.T) {
	r, _ := Check(base())
	got := map[string]Resolution{}
	for _, e := range r.Elements {
		got[e.ElementID] = e
	}
	if got["a"].What != "observed" || got["a"].State != StateDefined {
		t.Fatalf("%+v", got["a"])
	}
	if got["b"].What != "not-observed" || got["b"].State != StateDefined {
		t.Fatalf("%+v", got["b"])
	}
	if got["d"].What != "not-observed" || got["d"].State != StateUnfilled {
		t.Fatalf("%+v", got["d"])
	}
}

func TestAmbiguousRejected(t *testing.T) {
	in := base()
	in.Elements[2].Mappings = []Mapping{acc("d1"), acc("d2")}
	r, _ := Check(in)
	if r.Status != StatusRejected || len(r.Ambiguous) != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestCompleteAndIncomplete(t *testing.T) {
	in := base()
	r, _ := Check(in)
	if r.Status != StatusIncomplete {
		t.Fatal(r.Status)
	}
	in.Elements[2].Mappings = []Mapping{acc("d1")}
	in.Elements[3].Mappings = []Mapping{acc("d2")}
	r, _ = Check(in)
	if r.Status != StatusComplete {
		t.Fatal(r.Status)
	}
}

func TestValidation(t *testing.T) {
	cases := map[string]func(*Input){
		"unknown definition": func(i *Input) { i.Elements[0].Mappings[0].Definition = "zz" },
		"accepted needs":     func(i *Input) { i.Elements[0].Mappings[0].AcceptedBy = "" },
		"bad origin":         func(i *Input) { i.Elements[0].Mappings[0].Origin = "bot" },
		"bad status":         func(i *Input) { i.Elements[0].Mappings[0].Status = "ok" },
		"dup element":        func(i *Input) { i.Elements[1].ID = "a" },
		"dup definition":     func(i *Input) { i.Definitions[1].ID = "d1" },
		"dup mapping":        func(i *Input) { i.Elements[0].Mappings = append(i.Elements[0].Mappings, acc("d1")) },
		"schema":             func(i *Input) { i.Schema = "x" },
	}
	for name, mutate := range cases {
		in := base()
		mutate(&in)
		if _, err := Check(in); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestImpactWithinScope(t *testing.T) {
	before := base()
	after := base()
	after.Definitions = append(after.Definitions, Definition{ID: "d3"})
	after.Elements[2].Mappings = []Mapping{acc("d3")}
	imp, err := CompareImpact(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Status != ImpactWithinScope || len(imp.Changed) != 1 || imp.Changed[0].ElementID != "c" {
		t.Fatalf("%+v", imp)
	}
}

func TestImpactOutsideScope(t *testing.T) {
	before := base()
	after := base()
	after.Definitions = append(after.Definitions, Definition{ID: "d3"})
	after.Elements[2].Mappings = []Mapping{acc("d3")}
	// element b is rebound to an existing definition: change not caused by d3
	after.Elements[1].Mappings = []Mapping{acc("d2")}
	imp, err := CompareImpact(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if imp.Status != ImpactRejected || strings.Join(imp.Outside, ",") != "b" {
		t.Fatalf("%+v", imp)
	}
}

func TestImpactCandidateDoesNotChange(t *testing.T) {
	before := base()
	after := base()
	after.Elements[2].Mappings = []Mapping{{Definition: "d2", Status: MappingCandidate, Origin: OriginAI}}
	imp, err := CompareImpact(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(imp.Changed) != 0 || imp.Status != ImpactWithinScope {
		t.Fatalf("%+v", imp)
	}
}

func TestImpactRejectsStructuralChange(t *testing.T) {
	before := base()
	removed := base()
	removed.Definitions = removed.Definitions[:1]
	removed.Elements[0].Mappings = []Mapping{acc("d1")}
	if _, err := CompareImpact(before, removed); err == nil {
		t.Fatal("definition removal must be rejected")
	}
	fewer := base()
	fewer.Elements = fewer.Elements[:3]
	if _, err := CompareImpact(before, fewer); err == nil {
		t.Fatal("element set change must be rejected")
	}
	renamed := base()
	renamed.Elements[3].ID = "z"
	if _, err := CompareImpact(before, renamed); err == nil {
		t.Fatal("element id change must be rejected")
	}
}
