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
