package businessprocedure

import "testing"

// The declared vocabularies are contract values: a reader branches on them. This
// test names each one so the words the package writes are the words a test has
// seen, rather than values that only appear in a source file.
func TestDeclaredVocabulariesAreTheWordsThisTestNames(t *testing.T) {
	// Every realization, in report order.
	want := []Realization{RealizationManual, RealizationApplication, RealizationSystem}
	if len(realizations) != len(want) {
		t.Fatalf("realizations has %d entries, want %d", len(realizations), len(want))
	}
	for i, realization := range realizations {
		if realization != want[i] {
			t.Fatalf("realizations[%d]=%q, want %q", i, realization, want[i])
		}
		if !realization.Valid() {
			t.Fatalf("declared realization %q is not valid", realization)
		}
	}
	if Realization("workflow").Valid() {
		t.Fatal("an undeclared realization was accepted")
	}

	// The branch outcomes are a closed pair; a free string would let two
	// spellings of one decision look like two unrelated branches.
	if len(outcomes) != 2 || outcomes[0] != OutcomeApproval || outcomes[1] != OutcomeRejection {
		t.Fatalf("outcomes=%v, want approval then rejection", outcomes)
	}
	if !OutcomeApproval.Valid() || !OutcomeRejection.Valid() {
		t.Fatal("a declared outcome is not valid")
	}
	if Outcome("Rejected").Valid() {
		t.Fatal("an undeclared outcome was accepted")
	}

	// A reference fixture may only propose. "candidate" is the whole vocabulary,
	// so a fixture cannot write "accepted" and have it read as accepted.
	if len(statuses) != 1 || statuses[0] != StatusCandidate {
		t.Fatalf("statuses=%v, want only candidate", statuses)
	}
	if !StatusCandidate.Valid() {
		t.Fatal("the candidate status is not valid")
	}
	for _, rejected := range []Status{"accepted", "approved", "canonical"} {
		if rejected.Valid() {
			t.Fatalf("%q was accepted as a reference status", rejected)
		}
	}

	// Every reference kind reaches a Source Index subject kind, so a reference
	// cannot name a target the index cannot locate.
	for _, kind := range []ReferenceKind{
		ReferenceCapability, ReferenceValidation, ReferencePattern, ReferenceData,
	} {
		if _, ok := kind.subjectKind(); !ok {
			t.Fatalf("reference kind %q has no Source Index subject kind", kind)
		}
	}
	if _, ok := ReferenceKind("workflow").subjectKind(); ok {
		t.Fatal("an undeclared reference kind has a subject kind")
	}
	// The inverse table is derived from the same map, so the two directions
	// cannot disagree.
	for kind, subjectKind := range referenceSubjectKinds {
		back, ok := referenceKindForSubject(subjectKind)
		if !ok || back != kind {
			t.Fatalf("%q -> %q -> %q, want a round trip", kind, subjectKind, back)
		}
	}
	if _, ok := referenceKindForSubject("not-a-subject-kind"); ok {
		t.Fatal("an unknown subject kind mapped to a reference kind")
	}

	// The two severities are distinct and both are reported.
	if SeverityProcedure == SeverityReference {
		t.Fatal("the two severities are the same value")
	}
	projection := projectFixture(t, readFixture(t))
	if len(projection.Diagnostics) != 0 {
		t.Fatalf("the example produced findings: %+v", projection.Diagnostics)
	}

	// The schema versions name the input document and the output projection.
	if FixtureSchemaVersion != "business-procedure-model-reference-v0" {
		t.Fatalf("FixtureSchemaVersion=%q", FixtureSchemaVersion)
	}
	if ProjectionSchemaVersion != "tei.business-procedure.projection/v0" {
		t.Fatalf("ProjectionSchemaVersion=%q", ProjectionSchemaVersion)
	}
	if FixtureArtifactRole != "reference-authoring-fixture" {
		t.Fatalf("FixtureArtifactRole=%q", FixtureArtifactRole)
	}
	if EffectReadOnly != "read-only" {
		t.Fatalf("EffectReadOnly=%q", EffectReadOnly)
	}
}
