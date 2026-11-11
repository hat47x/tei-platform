package reacandidate

import (
	"strings"
	"testing"
)

// The carrier audit is the Fit/Gap answer for Issue #3394's exit criterion:
// every fact the minimal attempt carries is either mapped to an existing
// read-only type or named as a gap.
func TestAuditCarrierFitForMinimalAttempt(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	audit := AuditCarrierFit(set)
	if audit.SchemaVersion != candidateCarrierSchemaVers || audit.Effect != EffectReadOnly {
		t.Fatalf("audit envelope = %q %q", audit.SchemaVersion, audit.Effect)
	}
	// The literal is what a caller branches on, so pin it and not only the
	// constant. Otherwise renaming the constant silently renames the contract.
	if audit.SchemaVersion != "tei.rea.candidate-carrier-audit/v0" {
		t.Fatalf("carrier schemaVersion = %q", audit.SchemaVersion)
	}
	carried := map[string]CarrierFit{}
	for _, fit := range audit.Carried {
		carried[fit.Fact] = fit
	}
	missing := map[string]CarrierFit{}
	for _, fit := range audit.Missing {
		missing[fit.Fact] = fit
	}
	for _, fact := range []string{
		"source-artifact-path", "source-artifact-digest", "source-span",
		"relation-direction", "relation-label", "suggested-display-name",
		"subject-digest", "canonical-subject-resolution",
	} {
		fit, ok := carried[fact]
		if !ok {
			t.Fatalf("fact %q has no existing destination", fact)
		}
		if fit.Present == 0 {
			t.Fatalf("fact %q is carried but never present", fact)
		}
		if fit.Destination == "" {
			t.Fatalf("fact %q is listed as carried without a destination", fact)
		}
	}
	for _, fact := range []string{
		"external-node-identity", "external-observation-identity", "external-edge-identity",
		"external-endpoint-identities", "external-node-kind", "observed-state", "observed-authority",
		"evidence-coverage", "attempt-evidence-id", "provider-descriptor",
		"attempt-qualification", "provider-unknowns",
	} {
		fit, ok := missing[fact]
		if !ok {
			t.Fatalf("fact %q is missing from the gap list", fact)
		}
		if fit.Gap == "" {
			t.Fatalf("gap %q has no explanation", fact)
		}
	}
	// The audit must name both halves of the resolution path: the existing
	// operation can resolve a Canonical subject, and an external occurrence
	// cannot be resolved without an Owner mapping.
	resolution := carried["canonical-subject-resolution"]
	if !strings.Contains(resolution.Destination, "operations.InspectSubject") {
		t.Fatalf("destination = %q", resolution.Destination)
	}
	if !strings.Contains(resolution.Gap, "Owner mapping") {
		t.Fatalf("gap = %q", resolution.Gap)
	}
}

// The audit reports presence per fact, so an absent span is visible rather
// than silently mapped, and a kind-specific fact counts only the candidates it
// applies to.
func TestAuditCarrierFitReportsPresencePerFact(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	dataCount := len(candidatesOfKind(set, CandidateKindDataElement))
	relationCount := len(candidatesOfKind(set, CandidateKindCrossReference))
	spanCount := 0
	coverageCount := 0
	authorityCount := 0
	for _, candidate := range set.Candidates {
		if candidate.SourceRevision.Span != nil {
			spanCount++
		}
		if candidate.Facts.EvidenceCoverage != "" {
			coverageCount++
		}
		if candidate.Facts.ObservedAuthority != "" {
			authorityCount++
		}
	}
	audit := AuditCarrierFit(set)
	want := map[string]int{
		"source-artifact-path":          len(set.Candidates),
		"source-artifact-digest":        len(set.Candidates),
		"source-span":                   spanCount,
		"external-node-identity":        dataCount,
		"external-observation-identity": len(set.Candidates),
		"external-edge-identity":        relationCount,
		"relation-direction":            relationCount,
		"relation-label":                relationCount,
		"external-node-kind":            dataCount,
		"external-endpoint-identities":  relationCount,
		"observed-state":                len(set.Candidates),
		"observed-authority":            authorityCount,
		"evidence-coverage":             coverageCount,
		"suggested-display-name":        dataCount,
	}
	seen := map[string]bool{}
	for _, fit := range append(append([]CarrierFit{}, audit.Carried...), audit.Missing...) {
		seen[fit.Fact] = true
		if fit.Present > fit.Total {
			t.Fatalf("fact %q present %d > total %d", fit.Fact, fit.Present, fit.Total)
		}
		if expected, ok := want[fit.Fact]; ok && fit.Present != expected {
			t.Fatalf("fact %q present = %d, want %d", fit.Fact, fit.Present, expected)
		}
	}
	for fact := range want {
		if !seen[fact] {
			t.Fatalf("fact %q is not audited", fact)
		}
	}
	if len(audit.MissingFacts()) != len(audit.Missing) {
		t.Fatalf("MissingFacts = %v", audit.MissingFacts())
	}
	for _, line := range audit.Describe() {
		if !strings.Contains(line, ": ") {
			t.Fatalf("describe line = %q", line)
		}
	}
}
