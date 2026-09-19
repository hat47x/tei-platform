package main

import (
	"reflect"
	"sort"
	"testing"
)

type gateCSecondApplicationEntryInput struct {
	GateCStatus                string   `json:"gateCStatus"`
	HandoverBaselineID         string   `json:"handoverBaselineId"`
	BaselineRepositoryRevision string   `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string   `json:"targetBuildRevision"`
	SemanticEvidenceRef        string   `json:"semanticEvidenceRef"`
	FunctionalEvidenceRef      string   `json:"functionalEvidenceRef"`
	VisualEvidenceRef          string   `json:"visualEvidenceRef"`
	TargetDifferences          []string `json:"targetDifferences"`
	Unresolved                 []string `json:"unresolved,omitempty"`
}

type gateCSecondApplicationEntryQualification struct {
	State           string   `json:"state"`
	Missing         []string `json:"missing,omitempty"`
	AuthorityLimits []string `json:"authorityLimits"`
}

func TestGateCSecondApplicationEntryPreflightRequiresExplicitGateAndAllEvidenceClasses(t *testing.T) {
	got := qualifyGateCSecondApplicationEntry(gateCSecondApplicationEntryInput{
		GateCStatus:                "ready",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "e5231839a5dcc62cce1181a2731fcb7c70d16736",
		TargetBuildRevision:        "t1-test-only",
		SemanticEvidenceRef:        "evidence:semantic",
		FunctionalEvidenceRef:      "evidence:functional",
		VisualEvidenceRef:          "evidence:visual",
		TargetDifferences: []string{
			"web-target-differences-recorded",
		},
	})
	if got.State != "entry-evidence-complete" {
		t.Fatalf("complete Gate C entry evidence was not recognized: %+v", got)
	}
	if len(got.Missing) != 0 {
		t.Fatalf("complete entry unexpectedly has missing fields: %+v", got.Missing)
	}
	assertGateCSecondApplicationEntryNoAuthorityShortcut(t, reflect.TypeOf(gateCSecondApplicationEntryQualification{}))
}

func TestGateCSecondApplicationEntryPreflightBlocksCurrentPreTargetState(t *testing.T) {
	got := qualifyGateCSecondApplicationEntry(gateCSecondApplicationEntryInput{
		GateCStatus:                "not-ready",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "e5231839a5dcc62cce1181a2731fcb7c70d16736",
	})
	if got.State != "blocked" {
		t.Fatalf("pre-target state must remain blocked: %+v", got)
	}
	for _, want := range []string{
		"gate-c-ready",
		"target-build-revision",
		"semantic-evidence",
		"functional-evidence",
		"visual-evidence",
		"target-differences",
	} {
		if !containsGateCEntry(got.Missing, want) {
			t.Fatalf("pre-target state did not report missing %q: %+v", want, got.Missing)
		}
	}
}

func TestGateCSecondApplicationEntryPreflightRejectsVisualOnlyReadiness(t *testing.T) {
	got := qualifyGateCSecondApplicationEntry(gateCSecondApplicationEntryInput{
		GateCStatus:                "ready",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "e5231839a5dcc62cce1181a2731fcb7c70d16736",
		TargetBuildRevision:        "t1-test-only",
		VisualEvidenceRef:          "evidence:visual",
		TargetDifferences:          []string{"web-target-differences-recorded"},
	})
	if got.State != "blocked" {
		t.Fatalf("visual-only evidence must not open second-Application entry: %+v", got)
	}
	for _, want := range []string{"semantic-evidence", "functional-evidence"} {
		if !containsGateCEntry(got.Missing, want) {
			t.Fatalf("visual-only case lost missing %q: %+v", want, got.Missing)
		}
	}
}

func TestGateCSecondApplicationEntryPreflightRejectsReadyLabelWithoutBaselineOrTarget(t *testing.T) {
	got := qualifyGateCSecondApplicationEntry(gateCSecondApplicationEntryInput{
		GateCStatus:           "ready",
		SemanticEvidenceRef:   "evidence:semantic",
		FunctionalEvidenceRef: "evidence:functional",
		VisualEvidenceRef:     "evidence:visual",
		TargetDifferences:     []string{"web-target-differences-recorded"},
	})
	if got.State != "blocked" {
		t.Fatalf("ready label alone must not open second-Application entry: %+v", got)
	}
	for _, want := range []string{
		"handover-baseline",
		"baseline-repository-revision",
		"target-build-revision",
	} {
		if !containsGateCEntry(got.Missing, want) {
			t.Fatalf("ready-label-only case lost missing %q: %+v", want, got.Missing)
		}
	}
}

func TestGateCSecondApplicationEntryPreflightRequiresTargetDifferenceRecord(t *testing.T) {
	got := qualifyGateCSecondApplicationEntry(gateCSecondApplicationEntryInput{
		GateCStatus:                "ready",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: "e5231839a5dcc62cce1181a2731fcb7c70d16736",
		TargetBuildRevision:        "t1-test-only",
		SemanticEvidenceRef:        "evidence:semantic",
		FunctionalEvidenceRef:      "evidence:functional",
		VisualEvidenceRef:          "evidence:visual",
	})
	if got.State != "blocked" || !containsGateCEntry(got.Missing, "target-differences") {
		t.Fatalf("missing target-difference record must block entry: %+v", got)
	}
}

func qualifyGateCSecondApplicationEntry(in gateCSecondApplicationEntryInput) gateCSecondApplicationEntryQualification {
	var missing []string
	if in.GateCStatus != "ready" {
		missing = append(missing, "gate-c-ready")
	}
	if in.HandoverBaselineID == "" {
		missing = append(missing, "handover-baseline")
	}
	if in.BaselineRepositoryRevision == "" {
		missing = append(missing, "baseline-repository-revision")
	}
	if in.TargetBuildRevision == "" {
		missing = append(missing, "target-build-revision")
	}
	if in.SemanticEvidenceRef == "" {
		missing = append(missing, "semantic-evidence")
	}
	if in.FunctionalEvidenceRef == "" {
		missing = append(missing, "functional-evidence")
	}
	if in.VisualEvidenceRef == "" {
		missing = append(missing, "visual-evidence")
	}
	if len(in.TargetDifferences) == 0 {
		missing = append(missing, "target-differences")
	}
	sort.Strings(missing)

	state := "blocked"
	if len(missing) == 0 {
		state = "entry-evidence-complete"
	}
	return gateCSecondApplicationEntryQualification{
		State:   state,
		Missing: missing,
		AuthorityLimits: []string{
			"entry-evidence-complete-does-not-create-application-source",
			"entry-evidence-complete-does-not-imply-shared-acceptance",
			"gate-c-status-is-external-authority-input-not-derived-here",
			"visual-evidence-does-not-substitute-semantic-or-functional-evidence",
		},
	}
}

func containsGateCEntry(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertGateCSecondApplicationEntryNoAuthorityShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"AutoStart",
		"CreateApplication",
		"Accepted",
		"Approved",
		"CanonicalApplicationCreated",
		"SourceAuthoringAttempted",
		"EnvironmentBindingCreated",
		"TargetImplementationStarted",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("Gate C entry qualification introduced unsupported authority shortcut field %q", forbidden)
		}
	}
}
