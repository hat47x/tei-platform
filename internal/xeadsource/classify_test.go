package xeadsource

import (
	"reflect"
	"testing"
)

func TestClassifyRequiresExplicitPolicyAndKeepsUnknownUnresolved(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table/><FutureThing/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, ClassificationPolicy{
		"System": {Classifications: []string{"realization", "source-metadata"}, Extension: "reference.compat.xead.source"},
		"Table":  {Classifications: []string{"unresolved", "canonical-candidate"}, Extension: "reference.xead.relational-semantics"},
	})
	if len(got) != 3 {
		t.Fatalf("candidate count = %d", len(got))
	}
	if !reflect.DeepEqual(got[1].Classifications, []string{"canonical-candidate", "unresolved"}) {
		t.Fatalf("classifications must be deterministic: %#v", got[1])
	}
	if !got[2].Unresolved || len(got[2].Classifications) != 0 || got[2].Extension != "" {
		t.Fatalf("unknown source construct gained implicit meaning: %#v", got[2])
	}
}

func TestClassifyCarriesKeyAuthorityLimitsWithoutPromotingMeaning(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table><TableKey Type="PK"><TableKeyField FieldID="F1"/></TableKey></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"canonical-candidate", "source-key-assertion", "unresolved"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{
				"legacy-key-is-not-business-identity",
				"legacy-key-is-not-functional-dependency",
			},
		},
	})
	var key *Candidate
	for i := range got {
		if got[i].SourceName == "TableKey" {
			key = &got[i]
			break
		}
	}
	if key == nil {
		t.Fatal("TableKey candidate is missing")
	}
	if key.Unresolved {
		t.Fatalf("explicitly classified source construct remained unmapped: %#v", *key)
	}
	if !reflect.DeepEqual(key.Classifications, []string{"canonical-candidate", "source-key-assertion", "unresolved"}) {
		t.Fatalf("classifications drifted: %#v", *key)
	}
	if !reflect.DeepEqual(key.AuthorityLimits, []string{
		"legacy-key-is-not-business-identity",
		"legacy-key-is-not-functional-dependency",
	}) {
		t.Fatalf("authority limits drifted: %#v", *key)
	}
	if key.Extension != "reference.xead.relational-semantics" {
		t.Fatalf("extension drifted: %#v", *key)
	}
}

func TestClassificationPolicyFingerprintNormalizesRuleOrdering(t *testing.T) {
	a := ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"unresolved", "canonical-candidate"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{"z-limit", "a-limit"},
		},
	}
	b := ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"canonical-candidate", "unresolved"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{"a-limit", "z-limit"},
		},
	}
	if ClassificationPolicyFingerprint(a) == "" {
		t.Fatal("non-empty policy has no fingerprint")
	}
	if ClassificationPolicyFingerprint(a) != ClassificationPolicyFingerprint(b) {
		t.Fatalf("equivalent policy ordering changed fingerprint: a=%s b=%s",
			ClassificationPolicyFingerprint(a),
			ClassificationPolicyFingerprint(b),
		)
	}
	b["TableKey"] = CandidateRule{
		Classifications: []string{"canonical-candidate", "unresolved"},
		Extension:       "different.extension",
		AuthorityLimits: []string{"a-limit", "z-limit"},
	}
	if ClassificationPolicyFingerprint(a) == ClassificationPolicyFingerprint(b) {
		t.Fatal("different classification policy produced the same fingerprint")
	}
	if ClassificationPolicyFingerprint(nil) != "" {
		t.Fatal("nil policy must not claim classification provenance")
	}
}

func TestParseAloneDoesNotGrantCandidateMeaning(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Script/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := Classify(doc, nil)
	for _, candidate := range got {
		if !candidate.Unresolved || len(candidate.Classifications) != 0 || candidate.Extension != "" {
			t.Fatalf("source parsing granted semantic authority: %#v", candidate)
		}
	}
}
