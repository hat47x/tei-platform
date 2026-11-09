package main

import "testing"

func TestMetadataPreservingTransportKeepsReviewHistoryWhenAccessIsGranted(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "human-reviewed" {
			corpus.Assertions[i].AccessRefs = []string{"policy:project-tei"}
			corpus.Assertions[i].ReviewEvents[0].AccessRefs = []string{"policy:review-history"}
			corpus.Assertions[i].Evidence = []EvidenceRef{{
				Kind:       "source",
				Ref:        "repo://protected/source",
				AccessRefs: []string{"policy:evidence"},
			}}
		}
	}

	envelope, err := exportCorpusForAccess(corpus, TransportAccess{
		GrantedAccessRefs: []string{
			"policy:project-tei",
			"policy:review-history",
			"policy:evidence",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	imported, err := importTransportEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(imported, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "human-reviewed")
	if got.ConfirmationState != "confirmed" ||
		got.MetadataState != MetadataComplete ||
		got.ReviewBinding != "current" {
		t.Fatalf("metadata-preserving transport lost review authority: %+v", got)
	}
}

func TestRedactedReviewHistoryMakesImportedMetadataPartialAndFailClosed(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "human-reviewed" {
			corpus.Assertions[i].ReviewEvents[0].AccessRefs = []string{"policy:review-history"}
		}
	}

	envelope, err := exportCorpusForAccess(corpus, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Redactions.ReviewEvents != 1 {
		t.Fatalf("review redaction was not counted: %+v", envelope.Redactions)
	}
	imported, err := importTransportEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(imported, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "human-reviewed")
	if got.MetadataState != MetadataPartial ||
		got.ConfirmationState != "unreviewed" ||
		got.UseState != "review-required" ||
		!hasReason(got, "EPISTEMIC_METADATA_INCOMPLETE") {
		t.Fatalf("redacted review history was treated as complete: %+v", got)
	}
}

func TestInaccessibleAssertionDoesNotLeakThroughRelationsOrCoverageMapping(t *testing.T) {
	corpus, _ := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "conflict-b" {
			corpus.Assertions[i].AccessRefs = []string{"policy:restricted"}
		}
	}
	corpus.CoverageAreas = append(corpus.CoverageAreas, CoverageArea{
		ID:                 "restricted-conflict-coverage",
		Target:             TargetRef{Kind: "knowledge-area", ID: "tei.restricted.conflict"},
		Criticality:        CriticalityHigh,
		MappedAssertionIDs: []string{"conflict-b"},
	})

	envelope, err := exportCorpusForAccess(corpus, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Redactions.Assertions != 1 || envelope.Redactions.Relations == 0 ||
		envelope.Corpus.MetadataState != MetadataPartial {
		t.Fatalf("restricted assertion or relation was not redacted: envelope=%+v", envelope)
	}
	for _, assertion := range envelope.Corpus.Assertions {
		if assertion.ID == "conflict-b" {
			t.Fatal("restricted assertion leaked through transport")
		}
		for _, id := range append(append([]string{}, assertion.ConflictsWith...), assertion.Supersedes...) {
			if id == "conflict-b" {
				t.Fatal("restricted assertion identity leaked through visible relation")
			}
		}
	}
	for _, area := range envelope.Corpus.CoverageAreas {
		for _, id := range area.MappedAssertionIDs {
			if id == "conflict-b" {
				t.Fatal("restricted assertion identity leaked through coverage mapping")
			}
		}
	}
}

func TestEvidenceRedactionDoesNotSilentlyPreserveCompleteMetadata(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "ai-unreviewed" {
			corpus.Assertions[i].Evidence[0].AccessRefs = []string{"policy:restricted-evidence"}
		}
	}
	envelope, err := exportCorpusForAccess(corpus, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	imported, err := importTransportEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(imported, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "ai-unreviewed")
	if got.MetadataState != MetadataPartial ||
		!hasReason(got, "EPISTEMIC_METADATA_PARTIAL") ||
		!hasReason(got, "EPISTEMIC_METADATA_INCOMPLETE") {
		t.Fatalf("redacted Evidence remained epistemically complete: %+v", got)
	}
}

func TestTransportDoesNotInterpretAccessReferencesAsAuthority(t *testing.T) {
	corpus, _ := loadFixture(t)
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == "user-current" {
			corpus.Assertions[i].AccessRefs = []string{"opaque:any-existing-auth-policy-id"}
		}
	}
	if _, err := exportCorpusForAccess(corpus, TransportAccess{
		GrantedAccessRefs: []string{"opaque:different-policy-id"},
	}); err != nil {
		t.Fatal(err)
	}

	envelope, err := exportCorpusForAccess(corpus, TransportAccess{
		GrantedAccessRefs: []string{"opaque:any-existing-auth-policy-id"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, assertion := range envelope.Corpus.Assertions {
		if assertion.ID == "user-current" {
			found = true
		}
	}
	if !found {
		t.Fatal("exact opaque access reference was not honored")
	}
}

func TestHiddenOneSidedConflictMarksVisibleAssertionPartial(t *testing.T) {
	corpus, context := loadFixture(t)
	for i := range corpus.Assertions {
		switch corpus.Assertions[i].ID {
		case "conflict-a":
			corpus.Assertions[i].ConflictsWith = nil
		case "conflict-b":
			corpus.Assertions[i].ConflictsWith = []string{"conflict-a"}
			corpus.Assertions[i].AccessRefs = []string{"policy:restricted"}
		}
	}

	envelope, err := exportCorpusForAccess(corpus, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	imported, err := importTransportEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(imported, context)
	if err != nil {
		t.Fatal(err)
	}
	got := assessmentByID(t, projection, "conflict-a")
	if got.MetadataState != MetadataPartial ||
		got.UseState != "review-required" ||
		!hasReason(got, "EPISTEMIC_METADATA_INCOMPLETE") {
		t.Fatalf("hidden incoming conflict was silently forgotten: %+v", got)
	}
}

func TestPartialTransportHealthDoesNotClaimCompleteProjectView(t *testing.T) {
	corpus, context := loadFixture(t)
	corpus.Dependencies[0].AccessRefs = []string{"policy:restricted-dependency"}

	envelope, err := exportCorpusForAccess(corpus, TransportAccess{})
	if err != nil {
		t.Fatal(err)
	}
	imported, err := importTransportEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := evaluateCorpus(imported, context)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Health.ViewMetadataState != MetadataPartial {
		t.Fatalf("partial transport was presented as complete health view: %+v", projection.Health)
	}
	if !containsString(projection.AuthorityLimits, "partial-transport-health-is-not-project-wide-health") {
		t.Fatalf("missing partial-view health authority limit: %+v", projection.AuthorityLimits)
	}
}
