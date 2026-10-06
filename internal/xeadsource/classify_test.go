package xeadsource

import (
	"reflect"
	"testing"
)

func TestClassifyRequiresExplicitPolicyAndKeepsUnknownUnresolved(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table/><FutureThing/></System>`))
	if err != nil { t.Fatal(err) }
	got := Classify(doc, ClassificationPolicy{
		"System": {Classifications: []string{"realization", "source-metadata"}, Extension: "reference.compat.xead.source"},
		"Table":  {Classifications: []string{"unresolved", "canonical-candidate"}, Extension: "reference.xead.relational-semantics"},
	})
	if len(got) != 3 { t.Fatalf("candidate count = %d", len(got)) }
	if !reflect.DeepEqual(got[1].Classifications, []string{"canonical-candidate", "unresolved"}) {
		t.Fatalf("classifications must be deterministic: %#v", got[1])
	}
	if !got[2].Unresolved || len(got[2].Classifications) != 0 || got[2].Extension != "" {
		t.Fatalf("unknown source construct gained implicit meaning: %#v", got[2])
	}
}

func TestParseAloneDoesNotGrantCandidateMeaning(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Script/></System>`))
	if err != nil { t.Fatal(err) }
	got := Classify(doc, nil)
	for _, candidate := range got {
		if !candidate.Unresolved || len(candidate.Classifications) != 0 || candidate.Extension != "" {
			t.Fatalf("source parsing granted semantic authority: %#v", candidate)
		}
	}
}
