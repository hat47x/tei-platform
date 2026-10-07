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
	if got.RelationalResolutionPolicyRevision != RelationalResolutionPolicyRevision {
		t.Fatalf("resolution policy revision=%q", got.RelationalResolutionPolicyRevision)
	}
	if len(got.RelationalFactResolutions) != 1 ||
		got.RelationalFactResolutions[0].Kind != "legacy-key" ||
		got.RelationalFactResolutions[0].Status != RelationalFactResolutionUnresolved ||
		!reflect.DeepEqual(got.RelationalFactResolutions[0].DiagnosticCodes, []string{"key-field-not-found"}) {
		t.Fatalf("invalid key did not receive local unresolved status: %#v", got.RelationalFactResolutions)
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
		"relational-fact-resolution-is-structural-not-semantic-acceptance",
		"relational-resolution-policy-revision-is-algorithm-provenance-not-semantic-authority",
		"resolved-relational-fact-is-only-eligible-not-authorized-for-local-lowering",
		"unvalidated-relational-fact-blocks-deterministic-semantic-lowering",
		"unresolved-relational-resolution-blocks-full-set-deterministic-semantic-lowering",
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
	if got.RelationalResolutionStatus != "partial" ||
		len(got.RelationalFactResolutions) != 1 ||
		got.RelationalFactResolutions[0].Kind != "declared-key-fields" ||
		got.RelationalFactResolutions[0].Status != RelationalFactResolutionUnvalidated {
		t.Fatalf("unvalidated selection key was reported as lowerable: %#v", got.RelationalFactResolutions)
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
	if len(got.RelationalFactResolutions) != 1 ||
		got.RelationalFactResolutions[0].Status != RelationalFactResolutionResolved {
		t.Fatalf("valid key did not receive resolved local status: %#v", got.RelationalFactResolutions)
	}
}


func TestInspectKeepsResolvedFactsUsableWhenAnotherFactIsUnresolved(t *testing.T) {
	got, err := Inspect(
		FormatXead,
		[]byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey><TableKey ID="K2" Type="SK"><TableKeyField FieldID="MISSING"/></TableKey></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.RelationalResolutionStatus != RelationalFactResolutionUnresolved {
		t.Fatalf("full-set status did not remain unresolved: %#v", got)
	}
	statusByLocator := map[string]RelationalFactResolution{}
	for _, resolution := range got.RelationalFactResolutions {
		statusByLocator[resolution.Locator] = resolution
	}
	valid := statusByLocator["/System[1]/Table[1]/TableKey[2]"]
	if valid.Status != RelationalFactResolutionResolved ||
		len(valid.DiagnosticCodes) != 0 {
		t.Fatalf("unrelated valid key was blocked by another key: %#v", valid)
	}
	invalid := statusByLocator["/System[1]/Table[1]/TableKey[3]"]
	if invalid.Status != RelationalFactResolutionUnresolved ||
		!reflect.DeepEqual(invalid.DiagnosticCodes, []string{"key-field-not-found"}) {
		t.Fatalf("invalid key lost local diagnostic boundary: %#v", invalid)
	}
}

func TestInspectMarksAllDuplicateLegacyKeyOccurrencesUnresolved(t *testing.T) {
	got, err := Inspect(
		FormatXead,
		[]byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey><TableKey ID="K1" Type="SK"><TableKeyField FieldID="A"/></TableKey></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var duplicateKeyResolutions []RelationalFactResolution
	for _, resolution := range got.RelationalFactResolutions {
		if resolution.Kind == "legacy-key" {
			duplicateKeyResolutions = append(duplicateKeyResolutions, resolution)
		}
	}
	if len(duplicateKeyResolutions) != 2 {
		t.Fatalf("duplicate source key occurrences were lost: %#v", duplicateKeyResolutions)
	}
	for _, resolution := range duplicateKeyResolutions {
		if resolution.Status != RelationalFactResolutionUnresolved ||
			!reflect.DeepEqual(resolution.DiagnosticCodes, []string{"key-id-duplicate"}) {
			t.Fatalf("duplicate key occurrence was treated as locally resolved: %#v", resolution)
		}
	}
}


func TestInspectKeepsGlobalSourceIdentityDiagnosticUnresolvedWithoutRelationalFacts(t *testing.T) {
	got, err := Inspect(
		FormatXead,
		[]byte(`<System><Table><TableField/></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.RelationalFacts.LegacyKeys) != 0 ||
		len(got.RelationalFactResolutions) != 0 {
		t.Fatalf("fixture unexpectedly produced relational facts: %#v", got)
	}
	if got.RelationalResolutionStatus != RelationalFactResolutionUnresolved {
		t.Fatalf("global source identity diagnostics were hidden by empty fact set: %#v", got)
	}
	if len(got.RelationalDiagnostics) == 0 {
		t.Fatalf("missing table/field identity diagnostics were lost: %#v", got)
	}
}


func TestInspectPropagatesKeyMemberAmbiguityOnlyToDependentRelationship(t *testing.T) {
	got, err := Inspect(
		FormatXead,
		[]byte(`<System><Table ID="T1"><TableField ID="A"/><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey></Table><Table ID="T2"><TableField ID="B"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="B"/></TableKey></Table><Table ID="T3"><TableField ID="C"/><TableKey ID="K3" Type="PK"><TableKeyField FieldID="C"/></TableKey></Table><Table ID="T4"><TableField ID="D"/><TableKey ID="K4" Type="PK"><TableKeyField FieldID="D"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/><Relationship ID="R2" Table1ID="T3" TableKey1ID="K3" Table2ID="T4" TableKey2ID="K4"/></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	resolutionByLocator := map[string]RelationalFactResolution{}
	for _, resolution := range got.RelationalFactResolutions {
		resolutionByLocator[resolution.Locator] = resolution
	}
	var r1, r2 RelationalFactResolution
	for _, relation := range got.RelationalFacts.RelationshipKeys {
		switch relation.RelationshipID {
		case "R1":
			r1 = resolutionByLocator[relation.Locator]
		case "R2":
			r2 = resolutionByLocator[relation.Locator]
		}
	}
	if r1.Status != RelationalFactResolutionUnresolved {
		t.Fatalf("relationship using ambiguous key member was marked resolved: %#v", r1)
	}
	if !reflect.DeepEqual(r1.DiagnosticCodes, []string{"relationship-key-field-ambiguous"}) {
		t.Fatalf("dependent relationship lost propagated diagnostic: %#v", r1)
	}
	if r2.Status != RelationalFactResolutionResolved || len(r2.DiagnosticCodes) != 0 {
		t.Fatalf("unrelated relationship was blocked by another relation's ambiguity: %#v", r2)
	}
}
