package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

const (
	actorContextR3HGateInsufficientEvidence = "insufficient-evidence"
	actorContextR3HGateSyntheticOnly        = "synthetic-only-pressure"
	actorContextR3HGateQualified            = "exception-mechanism-pressure-qualified"
)

type actorContextR3HExceptionEvidenceProbe struct {
	Pattern               string
	PatternStatus         string
	PatternSource         string
	BusinessAddress       string
	BusinessStatus        string
	BusinessSource        string
	RepositoryBacked      bool
	PatternAuthorityKind  string
	BusinessAuthorityKind string
}

type actorContextR3HExceptionGateProbe struct {
	Classification  string
	BlockingReasons []string
	AuthorityLimits []string
}

func TestActorContextR3HCurrentReferenceHasNoExplicitNotRequiredExceptionCase(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range []struct {
		fixture fixture
		stepID  string
		want    string
	}{
		{fixture: primary, stepID: "BP-00", want: actorContextR3BAuthorizationUnresolved},
		{fixture: followUp, stepID: "FU-00", want: actorContextR3BAuthorizationUnresolved},
		{fixture: primary, stepID: "BP-06", want: actorContextR3BAuthorizationRequired},
	} {
		got, err := deriveActorContextR3BAuthorizationRequirementProbe(item.fixture, item.stepID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != item.want {
			t.Fatalf("current requirement state changed for %s/%s: got=%q want=%q probe=%+v",
				item.fixture.Procedure.ID, item.stepID, got.Status, item.want, got)
		}
	}

	for _, f := range []fixture{primary, followUp} {
		for _, s := range f.Steps {
			req, err := deriveActorContextR3BAuthorizationRequirementProbe(f, s.ID)
			if err != nil {
				t.Fatal(err)
			}
			if req.Status == actorContextR3BAuthorizationNotRequired {
				t.Fatalf("current Reference unexpectedly contains an explicit not-required exception candidate at %s/%s: %+v",
					f.Procedure.ID, s.ID, req)
			}
		}
	}
}

func TestActorContextR3HLegacyPolicyCannotQualifyPatternOrExceptionAuthority(t *testing.T) {
	got := actorContextR3HEvaluateExceptionGate(actorContextR3HExceptionEvidenceProbe{
		Pattern:               "order_entry.recent_orders",
		PatternStatus:         actorContextR3BAuthorizationRequired,
		PatternSource:         "legacy-policy:browse/order_entry.recent_orders",
		BusinessAddress:       "order-entry.follow-up/FU-00",
		BusinessStatus:        actorContextR3BAuthorizationNotRequired,
		BusinessSource:        "business-step:order-entry.follow-up/FU-00",
		RepositoryBacked:      true,
		PatternAuthorityKind:  "legacy-policy",
		BusinessAuthorityKind: "business-step",
	})

	if got.Classification != actorContextR3HGateInsufficientEvidence {
		t.Fatalf("Legacy policy was promoted to Pattern requirement authority: %+v", got)
	}
	if !reflect.DeepEqual(got.BlockingReasons, []string{"pattern-minimum-authority-unresolved"}) {
		t.Fatalf("unexpected policy-inference blocker set: %+v", got.BlockingReasons)
	}
}

func TestActorContextR3HSyntheticConflictDoesNotOpenProductionExceptionMechanism(t *testing.T) {
	got := actorContextR3HEvaluateExceptionGate(actorContextR3HExceptionEvidenceProbe{
		Pattern:               "order_entry.customer_reference",
		PatternStatus:         actorContextR3BAuthorizationRequired,
		PatternSource:         "test-only:pattern-minimum",
		BusinessAddress:       "customer-inquiry.customer-confirmation/CI-01",
		BusinessStatus:        actorContextR3BAuthorizationNotRequired,
		BusinessSource:        "test-only:business-exception",
		RepositoryBacked:      false,
		PatternAuthorityKind:  "application-pattern",
		BusinessAuthorityKind: "business-step",
	})

	if got.Classification != actorContextR3HGateSyntheticOnly {
		t.Fatalf("synthetic semantic pressure was promoted to repository-backed mechanism need: %+v", got)
	}
	if len(got.BlockingReasons) != 0 {
		t.Fatalf("synthetic pressure should be semantically complete but production-insufficient, got blockers=%v", got.BlockingReasons)
	}
}

func TestActorContextR3HRepositoryBackedExplicitConflictOnlyOpensDesignPressure(t *testing.T) {
	got := actorContextR3HEvaluateExceptionGate(actorContextR3HExceptionEvidenceProbe{
		Pattern:               "example.pattern",
		PatternStatus:         actorContextR3BAuthorizationRequired,
		PatternSource:         "application-pattern:example.pattern",
		BusinessAddress:       "example.procedure/EX-01",
		BusinessStatus:        actorContextR3BAuthorizationNotRequired,
		BusinessSource:        "business-step:example.procedure/EX-01",
		RepositoryBacked:      true,
		PatternAuthorityKind:  "application-pattern",
		BusinessAuthorityKind: "business-step",
	})

	if got.Classification != actorContextR3HGateQualified {
		t.Fatalf("fully explicit repository-backed conflict did not open exception-mechanism pressure: %+v", got)
	}
	if len(got.BlockingReasons) != 0 {
		t.Fatalf("qualified pressure unexpectedly retained blockers: %+v", got.BlockingReasons)
	}

	typ := reflect.TypeOf(got)
	for _, forbidden := range []string{
		"Winner",
		"Precedence",
		"Override",
		"EffectiveRequirement",
		"Authorized",
		"Allowed",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3H Evidence Gate introduced mechanism/runtime shortcut field %q", forbidden)
		}
	}
}

func actorContextR3HEvaluateExceptionGate(e actorContextR3HExceptionEvidenceProbe) actorContextR3HExceptionGateProbe {
	result := actorContextR3HExceptionGateProbe{
		AuthorityLimits: []string{
			"policy-does-not-author-pattern-minimum",
			"synthetic-pressure-does-not-justify-production-mechanism",
			"qualified-exception-pressure-does-not-select-precedence",
			"qualified-exception-pressure-does-not-select-override-syntax",
			"exception-applicability-does-not-establish-runtime-authorization",
		},
	}

	if e.Pattern == "" ||
		e.PatternStatus != actorContextR3BAuthorizationRequired ||
		e.PatternSource == "" ||
		e.PatternAuthorityKind != "application-pattern" {
		result.Classification = actorContextR3HGateInsufficientEvidence
		result.BlockingReasons = append(result.BlockingReasons, "pattern-minimum-authority-unresolved")
	}
	if e.BusinessAddress == "" ||
		e.BusinessStatus != actorContextR3BAuthorizationNotRequired ||
		e.BusinessSource == "" ||
		e.BusinessAuthorityKind != "business-step" {
		result.Classification = actorContextR3HGateInsufficientEvidence
		result.BlockingReasons = append(result.BlockingReasons, "business-exception-authority-unresolved")
	}
	if e.PatternSource != "" && e.PatternSource == e.BusinessSource {
		result.Classification = actorContextR3HGateInsufficientEvidence
		result.BlockingReasons = append(result.BlockingReasons, "distinct-provenance-unresolved")
	}

	result.BlockingReasons = actorContextR0SortedUnique(result.BlockingReasons)
	if len(result.BlockingReasons) > 0 {
		result.Classification = actorContextR3HGateInsufficientEvidence
		return result
	}
	if !e.RepositoryBacked {
		result.Classification = actorContextR3HGateSyntheticOnly
		return result
	}

	result.Classification = actorContextR3HGateQualified
	return result
}
