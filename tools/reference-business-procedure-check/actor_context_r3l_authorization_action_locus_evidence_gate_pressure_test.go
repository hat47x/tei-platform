package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3LGateInsufficientEvidence = "insufficient-evidence"
	actorContextR3LGateSingleQualifying      = "single-qualifying-interaction"
	actorContextR3LGateQualified             = "action-locus-pressure-qualified"
)

type actorContextR3LProtectedInteractionEvidenceProbe struct {
	BusinessAddress       string
	ApplicationPattern    string
	PatternKind           string
	RequirementStatus     string
	RequirementSource     string
	Resource              string
	ResourceSource        string
	Action                string
	ActionSource          string
	RepositoryBacked      bool
	ProviderIndependent   bool
	DiscriminatingFeature string
}

type actorContextR3LActionLocusGateProbe struct {
	Classification     string
	QualifyingAddresses []string
	BlockingReasons    []string
	AuthorityLimits    []string
}

func TestActorContextR3LCurrentRepositoryDoesNotQualifyActionLocusSelection(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	bp06Requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(primary, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	bp06Targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if !reflect.DeepEqual(bp06Targets, []string{"order_entry.submit_order"}) {
		t.Fatalf("test setup changed: BP-06 protected target=%v", bp06Targets)
	}
	bp06ActionGate := actorContextR3JDeriveActionAuthorityGate(
		bp06Requirement.Status,
		bp06Targets[0],
		"reference-pressure-overlay:BP-06",
		true,
		true,
		true,
		nil,
	)
	if bp06ActionGate.Action != "" ||
		bp06ActionGate.ActionAuthorityStatus != actorContextR3JActionAuthorityUnresolved {
		t.Fatalf("test setup changed: BP-06 action authority unexpectedly resolved: %+v", bp06ActionGate)
	}

	bp00Requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(primary, "BP-00")
	if err != nil {
		t.Fatal(err)
	}
	fu00Requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(followUp, "FU-00")
	if err != nil {
		t.Fatal(err)
	}

	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LProtectedInteractionEvidenceProbe{
		{
			BusinessAddress:       "order-entry.order-handling/BP-06",
			ApplicationPattern:    "order_entry.submit_order",
			PatternKind:           "command",
			RequirementStatus:     bp06Requirement.Status,
			RequirementSource:     bp06Requirement.RequirementSource,
			Resource:              bp06Targets[0],
			ResourceSource:        "reference-pressure-overlay:BP-06",
			Action:                bp06ActionGate.Action,
			ActionSource:          bp06ActionGate.ActionSource,
			RepositoryBacked:      true,
			ProviderIndependent:   true,
			DiscriminatingFeature: "command",
		},
		{
			BusinessAddress:       "order-entry.order-handling/BP-00",
			ApplicationPattern:    "order_entry.recent_orders",
			PatternKind:           "browse",
			RequirementStatus:     bp00Requirement.Status,
			RepositoryBacked:      true,
			ProviderIndependent:   false,
			DiscriminatingFeature: "browse-cross-procedure-reuse",
		},
		{
			BusinessAddress:       "order-entry.follow-up/FU-00",
			ApplicationPattern:    "order_entry.recent_orders",
			PatternKind:           "browse",
			RequirementStatus:     fu00Requirement.Status,
			RepositoryBacked:      true,
			ProviderIndependent:   false,
			DiscriminatingFeature: "browse-cross-procedure-reuse",
		},
	})

	if got.Classification != actorContextR3LGateInsufficientEvidence {
		t.Fatalf("current Repository incorrectly qualified action-locus selection: %+v", got)
	}
	if len(got.QualifyingAddresses) != 0 {
		t.Fatalf("current incomplete interactions became qualifying action-authority cases: %+v", got)
	}
	if !actorContextR0Contains(got.BlockingReasons, "second-qualifying-protected-interaction-unresolved") {
		t.Fatalf("current gate lost missing second-case blocker: %+v", got.BlockingReasons)
	}
}

func TestActorContextR3LOneCompleteCaseStillCannotSelectReusableLocus(t *testing.T) {
	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LProtectedInteractionEvidenceProbe{
		{
			BusinessAddress:       "example.procedure/EX-01",
			ApplicationPattern:    "example.command",
			PatternKind:           "command",
			RequirementStatus:     actorContextR3BAuthorizationRequired,
			RequirementSource:     "business-step:example.procedure/EX-01",
			Resource:              "example.command",
			ResourceSource:        "business-protected-target:example.procedure/EX-01",
			Action:                "execute",
			ActionSource:          "business-protected-action:example.procedure/EX-01",
			RepositoryBacked:      true,
			ProviderIndependent:   true,
			DiscriminatingFeature: "command",
		},
	})

	if got.Classification != actorContextR3LGateSingleQualifying {
		t.Fatalf("one complete protected interaction was overgeneralized into a locus rule: %+v", got)
	}
	if !reflect.DeepEqual(got.QualifyingAddresses, []string{"example.procedure/EX-01"}) {
		t.Fatalf("single qualifying interaction lost its address: %+v", got)
	}
}

func TestActorContextR3LTwoCasesNeedDiscriminatingVariation(t *testing.T) {
	base := []actorContextR3LProtectedInteractionEvidenceProbe{
		{
			BusinessAddress:       "example.procedure/EX-01",
			ApplicationPattern:    "example.command.one",
			PatternKind:           "command",
			RequirementStatus:     actorContextR3BAuthorizationRequired,
			RequirementSource:     "business-step:example.procedure/EX-01",
			Resource:              "example.command.one",
			ResourceSource:        "protected-target:EX-01",
			Action:                "execute",
			ActionSource:          "protected-action:EX-01",
			RepositoryBacked:      true,
			ProviderIndependent:   true,
			DiscriminatingFeature: "same-shape",
		},
		{
			BusinessAddress:       "example.procedure/EX-02",
			ApplicationPattern:    "example.command.two",
			PatternKind:           "command",
			RequirementStatus:     actorContextR3BAuthorizationRequired,
			RequirementSource:     "business-step:example.procedure/EX-02",
			Resource:              "example.command.two",
			ResourceSource:        "protected-target:EX-02",
			Action:                "execute",
			ActionSource:          "protected-action:EX-02",
			RepositoryBacked:      true,
			ProviderIndependent:   true,
			DiscriminatingFeature: "same-shape",
		},
	}

	noVariation := actorContextR3LEvaluateActionLocusGate(base)
	if noVariation.Classification != actorContextR3LGateInsufficientEvidence ||
		!actorContextR0Contains(noVariation.BlockingReasons, "locus-discriminating-variation-unresolved") {
		t.Fatalf("duplicated cases without variation incorrectly qualified locus comparison: %+v", noVariation)
	}

	withVariation := append([]actorContextR3LProtectedInteractionEvidenceProbe(nil), base...)
	withVariation[1].PatternKind = "browse"
	withVariation[1].Action = "browse"
	withVariation[1].DiscriminatingFeature = "browse-cross-procedure-reuse"

	qualified := actorContextR3LEvaluateActionLocusGate(withVariation)
	if qualified.Classification != actorContextR3LGateQualified ||
		len(qualified.BlockingReasons) != 0 {
		t.Fatalf("two complete repository-backed cases with variation did not open locus comparison pressure: %+v", qualified)
	}

	typ := reflect.TypeOf(qualified)
	for _, forbidden := range []string{
		"Winner",
		"PreferredLocus",
		"Confidence",
		"SelectedLocus",
		"Authorized",
		"Allowed",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3L Evidence Gate introduced selection/runtime shortcut field %q", forbidden)
		}
	}
}

func TestActorContextR3LPolicyDependentRecordCannotQualify(t *testing.T) {
	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LProtectedInteractionEvidenceProbe{
		{
			BusinessAddress:       "example.procedure/EX-01",
			ApplicationPattern:    "example.command",
			PatternKind:           "command",
			RequirementStatus:     actorContextR3BAuthorizationRequired,
			RequirementSource:     "business-step:example.procedure/EX-01",
			Resource:              "example.command",
			ResourceSource:        "protected-target:EX-01",
			Action:                "execute",
			ActionSource:          "legacy-policy:execute/example.command",
			RepositoryBacked:      true,
			ProviderIndependent:   false,
			DiscriminatingFeature: "command",
		},
	})

	if got.Classification != actorContextR3LGateInsufficientEvidence ||
		len(got.QualifyingAddresses) != 0 {
		t.Fatalf("provider-dependent policy action became Business/Application authority: %+v", got)
	}
}

func actorContextR3LEvaluateActionLocusGate(
	records []actorContextR3LProtectedInteractionEvidenceProbe,
) actorContextR3LActionLocusGateProbe {
	result := actorContextR3LActionLocusGateProbe{
		AuthorityLimits: []string{
			"requirement-must-be-explicit-authority",
			"protected-resource-must-have-explicit-provenance",
			"authorization-action-must-have-explicit-business-application-provenance",
			"legacy-policy-hints-do-not-fill-missing-authority",
			"one-qualifying-interaction-does-not-select-reusable-locus",
			"qualified-pressure-does-not-select-locus-winner",
		},
	}

	var features []string
	for _, record := range records {
		if record.RequirementStatus != actorContextR3BAuthorizationRequired ||
			record.RequirementSource == "" ||
			record.Resource == "" ||
			record.ResourceSource == "" ||
			record.Action == "" ||
			record.ActionSource == "" ||
			!record.RepositoryBacked ||
			!record.ProviderIndependent {
			continue
		}
		result.QualifyingAddresses = append(result.QualifyingAddresses, record.BusinessAddress)
		if record.DiscriminatingFeature != "" {
			features = append(features, record.DiscriminatingFeature)
		}
	}

	result.QualifyingAddresses = actorContextR0SortedUnique(result.QualifyingAddresses)
	features = actorContextR0SortedUnique(features)

	switch len(result.QualifyingAddresses) {
	case 0:
		result.Classification = actorContextR3LGateInsufficientEvidence
		result.BlockingReasons = []string{"second-qualifying-protected-interaction-unresolved"}
	case 1:
		result.Classification = actorContextR3LGateSingleQualifying
		result.BlockingReasons = []string{"second-qualifying-protected-interaction-unresolved"}
	default:
		if len(features) < 2 {
			result.Classification = actorContextR3LGateInsufficientEvidence
			result.BlockingReasons = []string{"locus-discriminating-variation-unresolved"}
		} else {
			result.Classification = actorContextR3LGateQualified
		}
	}

	sort.Strings(result.QualifyingAddresses)
	result.BlockingReasons = actorContextR0SortedUnique(result.BlockingReasons)
	return result
}
