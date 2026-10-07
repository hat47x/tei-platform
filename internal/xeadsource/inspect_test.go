package xeadsource

import (
	"reflect"
	"testing"
)

func TestInspectCombinesSourceFactsAndCandidatesWithoutGrantingAuthority(t *testing.T) {
	source := []byte(`<System><Table ID="T1"><TableKey ID="K1" Type="PK"><TableKeyField FieldID="F1"/></TableKey></Table><FutureThing/></System>`)
	got, err := Inspect(FormatXead, source, ClassificationPolicy{
		"TableKey": {
			Classifications: []string{"canonical-candidate", "source-key-assertion", "unresolved"},
			Extension:       "reference.xead.relational-semantics",
			AuthorityLimits: []string{
				"legacy-key-is-not-business-identity",
				"legacy-key-is-not-functional-dependency",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClassificationPolicyFingerprint == "" {
		t.Fatal("classified inspection did not expose policy fingerprint")
	}
	if got.Document.SHA256 == "" || !reflect.DeepEqual(got.Document.UnknownNames, []string{"FutureThing"}) {
		t.Fatalf("source document evidence is incomplete: %#v", got.Document)
	}
	if len(got.RelationalFacts.LegacyKeys) != 1 {
		t.Fatalf("relational source facts are missing: %#v", got.RelationalFacts)
	}
	if len(got.RelationalDiagnostics) != 1 ||
		got.RelationalDiagnostics[0].Code != "key-field-not-found" {
		t.Fatalf("missing field reference was not preserved as unresolved: %#v", got.RelationalDiagnostics)
	}
	if got.RelationalResolutionStatus != "unresolved" {
		t.Fatalf("relational inconsistency was marked reusable: %#v", got)
	}
	key := got.RelationalFacts.LegacyKeys[0]
	if key.KeyLegacyID != "K1" || key.KeyType != "PK" || key.Nature != SourceFactNature {
		t.Fatalf("legacy key fact drifted: %#v", key)
	}

	var keyCandidate *Candidate
	var futureCandidate *Candidate
	for i := range got.Candidates {
		switch got.Candidates[i].SourceName {
		case "TableKey":
			keyCandidate = &got.Candidates[i]
		case "FutureThing":
			futureCandidate = &got.Candidates[i]
		}
	}
	if keyCandidate == nil || keyCandidate.Unresolved {
		t.Fatalf("explicit source-key candidate is missing: %#v", got.Candidates)
	}
	if futureCandidate == nil || !futureCandidate.Unresolved ||
		len(futureCandidate.Classifications) != 0 ||
		len(futureCandidate.AuthorityLimits) != 0 {
		t.Fatalf("unknown source meaning was completed implicitly: %#v", futureCandidate)
	}
	if !reflect.DeepEqual(got.AuthorityLimits, []string{
		"source-inspection-is-not-canonical-authority",
		"source-fact-is-not-accepted-semantic-meaning",
		"candidate-classification-is-not-shared-acceptance",
		"unresolved-source-meaning-must-not-be-auto-completed",
		"relational-resolution-status-is-source-consistency-not-canonical-meaning",
		"unresolved-relational-resolution-blocks-deterministic-semantic-lowering",
	}) {
		t.Fatalf("inspection authority limits drifted: %#v", got.AuthorityLimits)
	}
}

func TestInspectWithoutPolicyKeepsAllCandidateMeaningUnresolved(t *testing.T) {
	got, err := Inspect(FormatXeaf, []byte(`<System><Function ID="F1" KeyFields="A;B;"/></System>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClassificationPolicyFingerprint != "" {
		t.Fatalf("policy-free inspection claimed policy provenance: %s", got.ClassificationPolicyFingerprint)
	}
	if len(got.RelationalFacts.DeclaredKeyFields) != 1 {
		t.Fatalf("declared key-field source fact is missing: %#v", got.RelationalFacts)
	}
	for _, candidate := range got.Candidates {
		if !candidate.Unresolved ||
			len(candidate.Classifications) != 0 ||
			candidate.Extension != "" ||
			len(candidate.AuthorityLimits) != 0 {
			t.Fatalf("policy-free inspection granted candidate meaning: %#v", candidate)
		}
	}
}


func TestInspectMarksRelationalFactsResolvedOnlyWithoutDiagnostics(t *testing.T) {
	got, err := Inspect(
		FormatXead,
		[]byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.RelationalDiagnostics) != 0 {
		t.Fatalf("valid relational source produced diagnostics: %#v", got.RelationalDiagnostics)
	}
	if got.RelationalResolutionStatus != "resolved" {
		t.Fatalf("diagnostic-free relational source was not marked resolved: %#v", got)
	}
}
