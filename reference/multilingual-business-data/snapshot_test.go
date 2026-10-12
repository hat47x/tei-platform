package multilingualbusinessdata

import (
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func example() (Snapshot, Key, Key) {
	field := modelref.Key{Scope: "catalog", Subject: subject.Subject{Kind: "attribute", ID: "catalog.product.description"}}
	ja := Key{Tenant: "store-a", Record: "product:P-100", Field: field, Locale: "ja-JP", Edition: "catalog"}
	en := ja
	en.Locale = "en-US"
	source := Revision{Key: ja, Revision: "ja-1", Text: "安全に保管してください。", Origin: "original", Review: "approved"}
	translated := Revision{
		Key: en, Revision: "en-1", Text: "Store safely.", Origin: "translation",
		BasedOn: &SourceRef{Key: ja, Revision: "ja-1"}, Review: "approved",
	}
	s := Snapshot{
		Model: modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: field}}},
		Revisions: []Revision{source, translated},
		Publications: []Publication{{Key: ja, Revision: "ja-1"}, {Key: en, Revision: "en-1"}},
	}
	return s, ja, en
}

func containsDiagnostic(ds []Diagnostic, code string) bool {
	for _, d := range ds {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestOriginalAndApprovedTranslationAreIndependentValues(t *testing.T) {
	s, ja, en := example()
	if ds := Validate(s); len(ds) != 0 {
		t.Fatalf("unexpected validation errors: %+v", ds)
	}
	if got := Select(s, ja); got.Diagnostic != "" || got.Text != "安全に保管してください。" || got.Revision != "ja-1" {
		t.Fatalf("original incorrectly selected: %+v", got)
	}
	if got := Select(s, en); got.Diagnostic != "" || got.Text != "Store safely." || got.Revision != "en-1" {
		t.Fatalf("translation incorrectly selected: %+v", got)
	}
	other := en
	other.Tenant = "store-b"
	if got := Select(s, other); got.Diagnostic != "missing-publication" || got.Text != "" {
		t.Fatalf("other tenant discovered data: %+v", got)
	}
}

func TestOriginalRevisionChangeInvalidatesPublishedTranslation(t *testing.T) {
	s, ja, en := example()
	s.Revisions = append(s.Revisions, Revision{
		Key: ja, Revision: "ja-2", Text: "冷暗所に保管してください。", Origin: "original", Review: "approved",
	})
	s.Publications[0].Revision = "ja-2"
	if ds := Validate(s); len(ds) != 0 {
		t.Fatalf("historical translated value should remain a valid record: %+v", ds)
	}
	if got := Select(s, ja); got.Diagnostic != "" || got.Revision != "ja-2" {
		t.Fatalf("new original not selected: %+v", got)
	}
	if got := Select(s, en); got.Diagnostic != "stale-translation" || got.Text != "" {
		t.Fatalf("old translation escaped source freshness check: %+v", got)
	}
	// A new approved translation refers to the current source revision.
	s.Revisions = append(s.Revisions, Revision{
		Key: en, Revision: "en-2", Text: "Keep in a cool, dark place.", Origin: "translation",
		BasedOn: &SourceRef{Key: ja, Revision: "ja-2"}, Review: "approved",
	})
	s.Publications[1].Revision = "en-2"
	if got := Select(s, en); got.Diagnostic != "" || got.Text != "Keep in a cool, dark place." {
		t.Fatalf("updated translation failed: %+v", got)
	}
}

func TestPublicationIsExplicitNotLatestRevision(t *testing.T) {
	s, _, en := example()
	s.Revisions = append(s.Revisions, Revision{
		Key: en, Revision: "en-999", Text: "Unapproved changed text", Origin: "translation",
		BasedOn: &SourceRef{Key: s.Publications[0].Key, Revision: "ja-1"}, Review: "candidate",
	})
	if got := Select(s, en); got.Diagnostic != "" || got.Revision != "en-1" {
		t.Fatalf("proposal became published by revision order: %+v", got)
	}
	s.Publications[1].Revision = "en-999"
	if ds := Validate(s); !containsDiagnostic(ds, "unapproved-publication") {
		t.Fatalf("missing approval requirement: %+v", ds)
	}
	if got := Select(s, en); got.Diagnostic != "invalid-snapshot" {
		t.Fatalf("candidate improperly published: %+v", got)
	}
}

func TestTranslationProposalRequiresCurrentSourceAndTarget(t *testing.T) {
	s, ja, en := example()
	p := TranslationProposal{
		Target: en, ExpectedTargetRevision: "en-1",
		Source: SourceRef{Key: ja, Revision: "ja-1"}, Text: "Store it safely.",
	}
	if got := CheckProposal(s, p); got != "candidate-current" {
		t.Fatalf("eligible proposal rejected: %s", got)
	}
	// No mutation happened: still the old published text.
	if got := Select(s, en); got.Revision != "en-1" || got.Text != "Store safely." {
		t.Fatalf("proposal wrote to authoritative data: %+v", got)
	}
	s.Revisions = append(s.Revisions, Revision{Key: ja, Revision: "ja-2", Text: "更新後の原文", Origin: "original", Review: "approved"})
	s.Publications[0].Revision = "ja-2"
	if got := CheckProposal(s, p); got != "source-changed" {
		t.Fatalf("stale source accepted: %s", got)
	}
	p.Source.Revision = "ja-2"
	if got := CheckProposal(s, p); got != "candidate-current" {
		t.Fatalf("updated source rejected: %s", got)
	}
	p.ExpectedTargetRevision = "unknown-target"
	if got := CheckProposal(s, p); got != "target-changed" {
		t.Fatalf("concurrent target changed unnoticed: %s", got)
	}
	p.ExpectedTargetRevision = "en-1"
	p.Source.Key.Tenant = "other-tenant"
	if got := CheckProposal(s, p); got != "invalid-proposal" {
		t.Fatalf("cross-tenant source was accepted: %s", got)
	}
}

func TestIndependentContractLanguageDoesNotExpireWithOtherVersion(t *testing.T) {
	s, ja, en := example()
	s.Revisions[1].Origin = "independent"
	s.Revisions[1].BasedOn = nil
	s.Revisions[1].Text = "Binding English agreement, separately drafted."
	s.Revisions = append(s.Revisions, Revision{
		Key: ja, Revision: "ja-2", Text: "独立した日本語の正式契約", Origin: "independent", Review: "approved",
	})
	s.Publications[0].Revision = "ja-2"
	if got := Select(s, en); got.Diagnostic != "" || got.Text != "Binding English agreement, separately drafted." {
		t.Fatalf("independent edition incorrectly invalidated: %+v", got)
	}
}

func TestDuplicateMissingOrCrossTenantSourceIsRejected(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(*Snapshot, Key, Key)
		code  string
	}{
		{"duplicate revision", func(s *Snapshot, _, _ Key) { s.Revisions = append(s.Revisions, s.Revisions[0]) }, "duplicate-revision"},
		{"duplicate publication", func(s *Snapshot, _, _ Key) { s.Publications = append(s.Publications, s.Publications[0]) }, "ambiguous-publication"},
		{"missing source", func(s *Snapshot, _, _ Key) { s.Revisions[1].BasedOn.Revision = "does-not-exist" }, "missing-source-revision"},
		{"foreign source", func(s *Snapshot, _, _ Key) { s.Revisions[1].BasedOn.Key.Tenant = "store-b" }, "invalid-translation-source"},
		{"unbound field", func(s *Snapshot, _, _ Key) { s.Model.Nodes = nil }, "unbound-attribute"},
		{"unsupported schema", func(s *Snapshot, _, _ Key) { s.Model.Schema = "unexpected" }, "unsupported-model"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, ja, en := example()
			tc.edit(&s, ja, en)
			if ds := Validate(s); !containsDiagnostic(ds, tc.code) {
				t.Fatalf("expected %s: %+v", tc.code, ds)
			}
			if got := Select(s, en); got.Diagnostic != "invalid-snapshot" {
				t.Fatalf("invalid snapshot was readable: %+v", got)
			}
		})
	}
}

func TestChainedTranslationCannotHideStaleIntermediate(t *testing.T) {
	s, ja, en := example()
	fr := ja
	fr.Locale = "fr-FR"
	s.Revisions = append(s.Revisions, Revision{
		Key: fr, Revision: "fr-1", Text: "Conserver avec soin.", Origin: "translation",
		BasedOn: &SourceRef{Key: en, Revision: "en-1"}, Review: "approved",
	})
	if ds := Validate(s); !containsDiagnostic(ds, "chained-translation-unsupported") {
		t.Fatalf("transitive dependency silently permitted: %+v", ds)
	}
}
