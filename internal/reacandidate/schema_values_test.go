package reacandidate

import "testing"

// Every response this package produces carries a schema identifier, and a reader
// branches on it. These identifiers were written by source without any test
// naming them, so each is generated here through the builder that writes it
// (#4032).
func TestResponseSchemasAreFixed(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())

	t.Run("candidate set", func(t *testing.T) {
		if set.SchemaVersion != "tei.rea.candidate-set/v0" {
			t.Fatalf("candidate set schema = %q", set.SchemaVersion)
		}
	})

	t.Run("candidate review", func(t *testing.T) {
		packet := BuildReviewPacket(set, nil, nil, nil)
		if packet.SchemaVersion != "tei.rea.candidate-review/v0" {
			t.Fatalf("candidate review schema = %q", packet.SchemaVersion)
		}
	})

	t.Run("candidate carrier audit", func(t *testing.T) {
		audit := AuditCarrierFit(set)
		if audit.SchemaVersion != "tei.rea.candidate-carrier-audit/v0" {
			t.Fatalf("carrier audit schema = %q", audit.SchemaVersion)
		}
	})

	t.Run("adoption preview", func(t *testing.T) {
		preview, err := PreviewAdoption(set, testModel())
		if err != nil {
			t.Fatal(err)
		}
		if preview.SchemaVersion != "tei.rea.adoption-preview/v0" {
			t.Fatalf("adoption preview schema = %q", preview.SchemaVersion)
		}
	})

	t.Run("candidate set comparison", func(t *testing.T) {
		comparison := CompareCandidateSets(set, set)
		if comparison.SchemaVersion != "tei.rea.candidate-set-comparison/v0" {
			t.Fatalf("candidate set comparison schema = %q", comparison.SchemaVersion)
		}
	})

	t.Run("candidate binding", func(t *testing.T) {
		result, err := BindCandidates(fakeReadOnlyModel{}, set, nil)
		if err != nil {
			t.Fatalf("binding refused: %v", err)
		}
		if result.SchemaVersion != "tei.rea.candidate-binding/v0" {
			t.Fatalf("binding schema = %q", result.SchemaVersion)
		}
	})
}
